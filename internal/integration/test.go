package integration

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"io/ioutil"
	"net/url"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"
	"syscall"
	"text/template"
	"time"

	"git.supremind.info/product/app/traffic/com/util"
	testClient "git.supremind.info/product/visionmind/test/event_test/internal/client"
	"git.supremind.info/product/visionmind/test/event_test/internal/com"
	"golang.org/x/sync/singleflight"
	"qiniupkg.com/x/log.v7"
)

type TestArgs struct {
	Verbose    bool
	CreateOnly bool
	Match      string
	matchAst   ast.Expr
	Region     string
	MaxChannel int
	Delete     bool
	AssetHost  string
}

type IIntegrationTest interface {
	CreateTask(caseMeta *com.MetaData, imgUrls, videoUrls []string) error
	StartTask(caseMeta *com.MetaData) error
	StopTask(caseMeta *com.MetaData) error
	Clean(caseMeta *com.MetaData) error
	NewEventClient(caseMeta *com.MetaData) testClient.IEventClient
}

type ReportTplData struct {
	StartTime   time.Time
	Soft        *Software
	ReportSlice ReportSlice
}

var ErrLabelMismatch = errors.New("label not matched")

type Test struct {
	*com.TestCommon
	TestArgs   *TestArgs
	Tester     IIntegrationTest
	token      atomic.Value
	group      singleflight.Group
	flightFunc func() (interface{}, error)
	ctx        context.Context
}

func (t *Test) ProcessCases(cases []string) (err error) {
	if len(cases) == 0 {
		err = errors.New("cases is null")
		log.Errorf("ProcessCases: %+v", err)
		return
	}

	globalCtx, cancelFunc := context.WithCancel(context.Background())
	t.ctx = globalCtx
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT)
		<-sigChan
		cancelFunc()
	}()

	//解析并检查是否包含不支持的表达式
	var matchAst ast.Expr
	if t.TestArgs.Match != "" {
		matchAst, err = parser.ParseExpr(t.TestArgs.Match)
		if err != nil {
			log.Errorf("parsing match expr failed: %s", err.Error())
			return err
		}
	}
	if _, err = match(matchAst, map[string]string{}); err != nil {
		log.Errorf("checking match expr failed: %s", err.Error())
		return err
	}
	t.TestArgs.matchAst = matchAst

	t.token.Store("")
	t.flightFunc = func() (interface{}, error) {
		newToken, err := AuthLogin(t.Config.AuthHost, t.Config.AuthUserName, t.Config.AuthPassword)
		if err != nil {
			return nil, err
		}
		t.token.Store(newToken)
		return newToken, nil
	}
	if t.Config.AuthHost != "" {
		token, err := AuthLogin(t.Config.AuthHost, t.Config.AuthUserName, t.Config.AuthPassword)
		if err != nil {
			return err
		}
		t.token.Store(token)
	}

	var (
		wg     sync.WaitGroup
		worker = t.Config.Worker
		caseCh = make(chan string, worker)

		reportCh   = make(chan Report, 5)
		reportOpen = make(chan struct{})
	)

	for i := 0; i < worker; i++ {
		wg.Add(1)
		go func() {
			t.processCase(caseCh, reportCh)
			wg.Done()
		}()
	}

	soft, err := GetSoftware(t.FlowClient)
	if err != nil {
		soft = &Software{}
	}

	reportData := ReportTplData{
		StartTime:   time.Now(),
		Soft:        soft,
		ReportSlice: ReportSlice{},
	}
	fileName := path.Join(fmt.Sprintf("./report_%s.md", time.Now().Format("20060102150405")))

	funcMap := template.FuncMap{
		"GetDurationStr":    GetDurationStr,
		"GerMdPreformatted": GerMdPreformatted,
	}
	tmpl, err := template.New("report").Funcs(funcMap).Parse(reportTemplate)
	if err != nil {
		log.Warnf("parsing template failed: %v", err)
		return err
	}

	go func() {
		for report := range reportCh {
			reportData.ReportSlice = append(reportData.ReportSlice, report)
			// 排序
			sort.Sort(reportData.ReportSlice)

			// 模板替换
			var content bytes.Buffer
			err = tmpl.Execute(&content, reportData)
			if err != nil {
				err = fmt.Errorf("tmplate Execute failed, err: %v", err)
				log.Errorf("%+v", err)
				return
			}

			err = ioutil.WriteFile(fileName, content.Bytes(), 0755)
			if err != nil {
				err = fmt.Errorf("ioutil.WriteFile(%s): %v", fileName, err)
				log.Errorf("%+v", err)
				return
			}
			log.Info("report has been written in ", fileName)
		}

		close(reportOpen)
	}()

	for _, item := range cases {
		caseCh <- item
	}

	close(caseCh)
	wg.Wait()

	close(reportCh)
	<-reportOpen

	log.Info("all goroutine finish")

	return
}

func (t *Test) processCase(caseCh chan string, reportCh chan Report) {
	for item := range caseCh {
		metaData, err := t.doProcessCase(item)
		if err == ErrLabelMismatch {
			continue
		}
		report := Report{
			MetaData: metaData,
		}
		if err != nil {
			report.ErrorMsg = err.Error()
		}

		reportCh <- report
	}
}

func (t *Test) doProcessCase(metadataPath string) (metaData com.MetaData, err error) {
	var (
		videoURLs, snapshotURLs []string
		dirpath                 = filepath.Dir(metadataPath)
	)

	metaData, err = GetMetaData(metadataPath)
	if err != nil {
		return
	}

	//match label
	matched, _ := match(t.TestArgs.matchAst, metaData.Case.Label)
	if !matched {
		return metaData, ErrLabelMismatch
	}

	if metaData.Files != nil {
		//video
		for _, video := range metaData.Files.Videos {
			videoRawUrl, err := url.Parse(video)
			if err != nil {
				return metaData, err
			}
			if t.TestArgs.AssetHost != "" {
				videoRawUrl.Host = t.TestArgs.AssetHost
			}
			videoRawUrl.RawQuery = videoRawUrl.Query().Encode()
			videoURLs = append(videoURLs, videoRawUrl.String())
		}

		//images
		//违法截图暂时还是先上传到fileserver
		//web上截图经由nginx转发时没有设置Host头，ingress暴露的服务访问不到(minio)
		for i, img := range metaData.Files.Images {
			if t.TestArgs.AssetHost != "" {
				snapshotRawUrl, err := url.Parse(img)
				if err != nil {
					return metaData, err
				}
				snapshotRawUrl.Host = t.TestArgs.AssetHost
				img = snapshotRawUrl.String()
			}
			log.Infof("downloading snapshot from %s", img)
			snapshotPath := filepath.Join(dirpath, fmt.Sprintf("%s_img_%d%s", metaData.Case.Name, i, path.Ext(img)))
			if !util.Exist(snapshotPath) {
				err = DownloadAsset(img, dirpath, filepath.Base(snapshotPath))
				if err != nil {
					err = errors.New("下载违法截图失败: " + err.Error())
					return
				}
			}

			snapshotURL, err := t.uploadFile(snapshotPath, metaData.Task.Type)
			if err != nil {
				return metaData, err
			}
			snapshotRawUrl, err := url.Parse(snapshotURL)
			if err != nil {
				return metaData, err
			}
			snapshotRawUrl.RawQuery = snapshotRawUrl.Query().Encode()
			snapshotURLs = append(snapshotURLs, snapshotRawUrl.String())
		}
	}

	err = t.Tester.CreateTask(&metaData, snapshotURLs, videoURLs)
	if err != nil {
		return
	}
	log.Infof("created taskId %s for case %s", metaData.Task.ID, metaData.Case.Name)

	if t.TestArgs.CreateOnly {
		return
	}

	err = t.Tester.StartTask(&metaData)
	if err != nil {
		return
	}

	//事件比较
	eventCli := t.Tester.NewEventClient(&metaData)
	events2validate := make([]com.EventData, len(metaData.Event))
	copy(events2validate, metaData.Event)
	ctx, cancelFunc := context.WithTimeout(t.ctx, time.Duration(t.Config.EventTimeout)*time.Minute)
	defer cancelFunc()

	for len(events2validate) > 0 {
	AGAIN:
		nextEvents, errIn := eventCli.NextEvents(ctx, &metaData, time.Duration(t.Config.EventDuration)*time.Second, t.token.Load().(string))
		if errIn != nil {
			log.Warnf("%s: tester.NextEvents: %+v", metaData.Case.Name, errIn)
			if errIn == testClient.ErrUnAuthed {
				_, err, _ = t.group.Do("token", t.flightFunc)
				if err != nil {
					return
				}
				goto AGAIN
			}
			//超时，任务失败
			if errIn == testClient.ErrCanceled {
				if err == nil {
					err = errors.New("未产生匹配事件")
				}
				break
			}
		} else {
			beforeMatch := len(events2validate)
			for i := 0; i < len(events2validate); i++ {
				if err = eventCli.Match(&events2validate[i], nextEvents); err == nil {
					events2validate = append(events2validate[:i], events2validate[i+1:]...)
					i--
				}
			}

			if len(nextEvents) > 0 {
				afterMatch := len(events2validate)
				log.Infof("%s: fetched %d events, %d matched, %d left", metaData.Case.Name, len(nextEvents), beforeMatch-afterMatch, afterMatch)
			}
		}
	}

	//停掉已经测完case的task
	stopErr := t.Tester.StopTask(&metaData)
	if stopErr != nil {
		log.Errorf("tester.StopTask(%s): %s", metaData.Task.ID, stopErr.Error())
	}

	if t.TestArgs.Delete {
		cleanErr := t.Tester.Clean(&metaData)
		if cleanErr != nil {
			log.Errorf("clean task %s failed", metaData.Task.ID)
		}
	}

	return
}

// 上传文件
func (t *Test) uploadFile(path string, prefix string) (url string, err error) {
	bytes, err := ReadFile(path)
	if err != nil {
		return
	}

	url, err = t.FileserverClient.Save(prefix+"_"+filepath.Base(path), bytes)
	if err != nil {
		log.Errorf("t.FileserverClient.Save(%s): %+v", path, err)
		return
	}

	return
}
