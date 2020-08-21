package generate

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"git.supremind.info/product/visionmind/test/event_test/internal/client"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	appUtil "git.supremind.info/product/app/traffic/com/util"
	"git.supremind.info/product/visionmind/com/flow"
	"git.supremind.info/product/visionmind/test/event_test/internal/com"
	"git.supremind.info/product/visionmind/test/event_test/internal/integration"
	"github.com/minio/minio-go/v6"
	"qiniupkg.com/x/log.v7"
)

//|违法类型 | case标识 | 违法车牌号 | 车辆类型 | 开始时间“播放器上的播放进度时间” | 结束时间“播放器上的播放进度时间” | 第一级目录/第二级目录/视频文件名称 |白/黑|晴/雨|头/尾|大/小|
const (
	CaseViolateTypeIdx = iota
	CaseIdIdx
	CasePlateNoIdx
	CaseVehicleTypeIdx
	CaseVideoStartIdx
	CaseVideoEndIdx
	CaseVideoPathIdx
	CaseTimeIdx
	CaseWeatherIdx
	CaseCamAngleIdx
	CaseAngleIdx

	CaseFieldLength
)

type Generate struct {
	basePath   string
	taskJson   map[string]*flow.Task
	reportDone chan struct{}
	failedChan chan FailedCase
	arg        *GenerateArg
	config     *com.Config
	s3Client   *minio.Client
	gitlabCli *client.GitlabClient
}

type FailedCase struct {
	CsvName  string
	CsvIndex int
	Msg      string
}

type FailedReport struct {
	StartAt   time.Time
	CaseNum   int
	FailedNum int
	Cases     map[string][]FailedCase
}

func NewCaseGenerate(arg *GenerateArg, config *com.Config) (*Generate, error) {
	gen := Generate{
		basePath:   arg.BasePath,
		taskJson:   make(map[string]*flow.Task, 1),
		reportDone: make(chan struct{}),
		failedChan: make(chan FailedCase, config.Worker),
		arg:        arg,
		config:     config,
	}

	if arg.Upload {
		if config.Git.GitLab == "" || config.Git.Repo == "" || config.Git.UserName == "" {
			log.Warnf("git config is empty, cases won't be uploaded")
		} else {
			gen.gitlabCli = client.NewGitLabClient(config)
		}
	}

	return &gen, nil
}

func (g *Generate) StartGenerate(ctx context.Context) error {

	var (
		wait     sync.WaitGroup
		caseChan = make(chan Case, g.config.Worker)
	)

	err := os.MkdirAll(filepath.Join(g.basePath, "video"), 0755)
	if err != nil {
		return err
	}

	err = os.RemoveAll(filepath.Join(g.basePath, "case"))
	if err != nil {
		return err
	}
	err = os.MkdirAll(filepath.Join(g.basePath, "case"), 0755)
	if err != nil {
		return err
	}

	if g.arg.Upload {
		g.s3Client, err = minio.New(g.config.Minio.Host, g.config.Minio.AccessKeyID, g.config.Minio.SecretAccessKey, g.config.Minio.UseSSL)
		if err != nil {
			return err
		}
		bucketExists, err := g.s3Client.BucketExists(g.config.Minio.Bucket)
		if err != nil {
			return err
		}
		if !bucketExists {
			err = g.s3Client.MakeBucket(g.config.Minio.Bucket, "")
			if err != nil {
				return err
			}
			err = g.s3Client.SetBucketPolicy(g.config.Minio.Bucket, getFullAccessPolicy(g.config.Minio.Bucket))
			if err != nil {
				return err
			}
		}
	}

	files, err := ioutil.ReadDir(g.basePath)
	if err != nil {
		return err
	}

	tpl, err := template.New("report").Parse(FailedTemplate)
	if err != nil {
		return err
	}
	failedReport := FailedReport{
		StartAt: time.Now(),
	}
	go g.logFailed(tpl, &failedReport)

	cases := []Case{}
	for _, file := range files {
		if !file.IsDir() {
			path := filepath.Join(g.basePath, file.Name())
			switch {
			case strings.HasSuffix(file.Name(), ".csv"):
				subCases, subNum, err := g.parseCSV(path)
				if err != nil {
					return err
				}
				cases = append(cases, subCases...)
				failedReport.CaseNum += subNum
			case strings.HasSuffix(file.Name(), ".json"):
				err = g.loadTaskToTaskJson(path)
				if err != nil {
					return err
				}
			}
		}
	}
	failedReport.CaseNum = len(cases)

	for i := 0; i < g.config.Worker; i++ {
		wait.Add(1)
		go func() {
			g.doGenerate(ctx, caseChan)
			wait.Done()
		}()
	}
	for _, c := range cases {
		caseChan <- c
	}
	close(caseChan)

	wait.Wait()
	close(g.failedChan)
	<-g.reportDone

	if failedReport.FailedNum != 0 || failedReport.CaseNum == 0 {
		err = fmt.Errorf("case generate failed, %d/%d cases failed", failedReport.FailedNum, failedReport.CaseNum)
		return err
	}

	if g.arg.Upload && g.gitlabCli != nil && !g.arg.CutOnly {
		var (
			commitMsg     = fmt.Sprintf("%s: generate cases %s", time.Now().Format("20060102150405"), g.arg.RemoteBasePath)
			newBranchName = fmt.Sprintf("case%s", time.Now().Format("20060102150405"))
			casePath      = filepath.Join(g.arg.BasePath, "case")
			gitBasePath   = filepath.Join("cases", g.arg.RemoteBasePath)
		)
		forkedRepoPath, err := g.ensureForkExists()
		if err != nil {
			return err
		}

		err = g.pushNewCases(forkedRepoPath, newBranchName, casePath, gitBasePath, commitMsg)
		if err != nil {
			return err
		}

		if g.arg.PR {
			prUrl, err := g.gitlabCli.CreatePR(context.Background(), g.config.Git.Repo, forkedRepoPath, newBranchName, "master", commitMsg)
			if err != nil {
				return err
			}
			log.Infof("PR created, see %s", prUrl)
		}
	}

	return nil
}

func (g *Generate) doGenerate(ctx context.Context, caseChan <-chan Case) {
	for case2Gen := range caseChan {
		select {
		case <-ctx.Done():
			g.failedChan <- FailedCase{
				CsvName:  case2Gen.csvName,
				CsvIndex: case2Gen.csvIndex,
				Msg:      "已取消",
			}
			continue
		default:
		}

		var (
			err error
		)

		//windows 文件名里不能包含冒号
		caseBaseName := fmt.Sprintf("%s_%s_%s_%s_%s", case2Gen.violateType, case2Gen.identity, case2Gen.plateNo, strings.Replace(case2Gen.startTime, ":", "-", -1), strings.Replace(case2Gen.endTime, ":", "-", -1))
		metaJsonPath := filepath.Join(g.basePath, "case", case2Gen.violateType, case2Gen.location,
			fmt.Sprintf("%s.json", caseBaseName))
		cuttedVideoPath := filepath.Join(g.basePath, "video", case2Gen.violateType, case2Gen.location,
			fmt.Sprintf("%s.mp4", caseBaseName))
		snapshotPath := filepath.Join(g.basePath, "pic", case2Gen.location+".jpg")
		if !g.arg.CutOnly && !appUtil.Exist(snapshotPath) {
			g.failedChan <- FailedCase{
				CsvName:  case2Gen.csvName,
				CsvIndex: case2Gen.csvIndex,
				Msg:      "对应任务截图不存在",
			}
			continue
		}

		if !appUtil.Exist(case2Gen.OriginVideoPath) {
			g.failedChan <- FailedCase{
				CsvName:  case2Gen.csvName,
				CsvIndex: case2Gen.csvIndex,
				Msg:      "对应违法视频不存在",
			}
			continue
		}

		task, ok := g.taskJson[fmt.Sprintf("%s_%s", case2Gen.violateType, case2Gen.location)]
		if !g.arg.CutOnly && !ok {
			g.failedChan <- FailedCase{
				CsvName:  case2Gen.csvName,
				CsvIndex: case2Gen.csvIndex,
				Msg:      "对应布控任务不存在",
			}
			continue
		}

		if g.arg.OverWrite || !appUtil.Exist(cuttedVideoPath) {
			outputArg := g.arg.Args
			if g.arg.RemoveBFrame {
				outputArg = RemoveBFrameOpt
			}
			err := CutVideo(ctx, case2Gen.OriginVideoPath, cuttedVideoPath, case2Gen.startTime, case2Gen.endTime, g.arg.Verbose, outputArg)
			if err != nil {
				g.failedChan <- FailedCase{
					CsvName:  case2Gen.csvName,
					CsvIndex: case2Gen.csvIndex,
					Msg:      err.Error(),
				}
				continue
			}
		}
		if g.arg.CutOnly {
			continue
		}

		if g.arg.Upload {
			case2Gen.remoteVideoPath, case2Gen.remoteSnapshotPath, err = g.upload(&case2Gen, cuttedVideoPath, snapshotPath)
			if err != nil {
				g.failedChan <- FailedCase{
					CsvName:  case2Gen.csvName,
					CsvIndex: case2Gen.csvIndex,
					Msg:      fmt.Sprintf("%s: %s", "上传视频和截图失败", err.Error()),
				}
				continue
			}
		}

		if g.arg.OverWrite || g.arg.Upload || !appUtil.Exist(metaJsonPath) {
			caseMeta, err := g.generateCaseMeta(&case2Gen, task)
			if err != nil {
				g.failedChan <- FailedCase{
					CsvName:  case2Gen.csvName,
					CsvIndex: case2Gen.csvIndex,
					Msg:      err.Error(),
				}
				continue
			}

			caseMetaContent, err := json.MarshalIndent(caseMeta, "", "\t")
			if err != nil {
				g.failedChan <- FailedCase{
					CsvName:  case2Gen.csvName,
					CsvIndex: case2Gen.csvIndex,
					Msg:      err.Error(),
				}
				continue
			}

			_ = os.MkdirAll(filepath.Dir(metaJsonPath), 0755)
			err = ioutil.WriteFile(metaJsonPath, caseMetaContent, 0755)
			if err != nil {
				g.failedChan <- FailedCase{
					CsvName:  case2Gen.csvName,
					CsvIndex: case2Gen.csvIndex,
					Msg:      err.Error(),
				}
				continue
			}
		}
	}
}

func (g *Generate) parseCSV(path string) ([]Case, int, error) {
	cases := []Case{}
	csvReader, err := os.OpenFile(path, os.O_RDONLY, 0755)
	if err != nil {
		return nil, 0, err
	}

	reader := csv.NewReader(csvReader)
	_, err = reader.Read()
	if err != nil {
		return nil, 0, err
	}
	if reader.FieldsPerRecord < CaseFieldLength {
		return nil, 0, fmt.Errorf("csv格式错误，列数小于%d", CaseFieldLength)
	}

	videoTimeRegex := regexp.MustCompile("^[0-9]+:[0-5][0-9]:[0-5][0-9]$")
	index := 1
	for {
		record, err := reader.Read()
		index++
		if err != nil {
			if err == io.EOF {
				return cases, index - 2, nil
			}
			g.failedChan <- FailedCase{
				CsvIndex: index,
				CsvName:  filepath.Base(path),
				Msg:      fmt.Sprintf("csv格式错误: %s", err.Error()),
			}
			continue
		}
		videoDirectories := strings.Split(record[CaseVideoPathIdx], "/")
		if len(videoDirectories) != 3 {
			g.failedChan <- FailedCase{
				CsvName:  filepath.Base(path),
				CsvIndex: index,
				Msg:      "视频位置格式不正确, 应为:第一级目录/第二级目录/视频文件名称",
			}
			continue
		}

		if (record[CaseVideoStartIdx] != "" || record[CaseVideoEndIdx] != "") && (!videoTimeRegex.Match([]byte(record[CaseVideoStartIdx])) || !videoTimeRegex.Match([]byte(record[CaseVideoEndIdx]))) {
			g.failedChan <- FailedCase{
				CsvName:  filepath.Base(path),
				CsvIndex: index,
				Msg:      "视频开始或结束时间格式错误，应为: 00:00:00",
			}
			continue
		}

		if strings.TrimSpace(record[CaseIdIdx]) == "" {
			g.failedChan <- FailedCase{
				CsvName:  filepath.Base(path),
				CsvIndex: index,
				Msg:      "case标识符为空",
			}
		}

		case2Gen := Case{
			identity:        record[CaseIdIdx],
			csvName:         path,
			csvIndex:        index,
			violateType:     record[CaseViolateTypeIdx],
			OriginVideoPath: filepath.Join(g.basePath, filepath.Join(videoDirectories...)),
			location:        videoDirectories[0],
			vehicleType:     record[CaseVehicleTypeIdx],
			plateNo:         record[CasePlateNoIdx],
			startTime:       record[CaseVideoStartIdx],
			endTime:         record[CaseVideoEndIdx],
			labels: map[string]string{
				"time":     record[CaseTimeIdx],
				"weather":  record[CaseWeatherIdx],
				"camAngle": record[CaseCamAngleIdx],
				"angle":    record[CaseAngleIdx],
			},
		}
		cases = append(cases, case2Gen)
	}
}

func (g *Generate) loadTaskToTaskJson(path string) error {
	fileContent, err := integration.ReadFile(path)
	if err != nil {
		return err
	}

	task := flow.Task{}
	err = json.Unmarshal(fileContent, &task)
	if err != nil {
		return err
	}

	g.taskJson[strings.TrimSuffix(filepath.Base(path), ".json")] = &task
	return nil
}

func (g *Generate) logFailed(tpl *template.Template, report *FailedReport) {
	defer close(g.reportDone)
	failedCases := make(map[string][]FailedCase)
	reportFileName := filepath.Join(g.basePath, fmt.Sprintf("failed_generate_%s.md", report.StartAt.Format("20060102150405")))
	for failedCase := range g.failedChan {
		if g.arg.Verbose {
			log.Infof("generate case failed, csv: %s, index: %d, msg: %s", failedCase.CsvName, failedCase.CsvIndex, failedCase.Msg)
		}
		failedCases[failedCase.CsvName] = append(failedCases[failedCase.CsvName], failedCase)
		sort.Slice(failedCases[failedCase.CsvName], func(i, j int) bool {
			return failedCases[failedCase.CsvName][i].CsvIndex < failedCases[failedCase.CsvName][j].CsvIndex
		})
		report.FailedNum++
	}

	report.Cases = failedCases
	var content bytes.Buffer
	err := tpl.Execute(&content, &report)
	if err != nil {
		log.Errorf("logFailed.tpl.Execute() failed: %s", err.Error())
		return
	}

	err = ioutil.WriteFile(reportFileName, content.Bytes(), 0755)
	if err != nil {
		log.Errorf("logFailed.ioutil.WriteFile() failed: %s", err.Error())
		return
	}
	log.Infof("failed list has been written to %s", reportFileName)
}

func (g *Generate) generateCaseMeta(case2Gen *Case, task *flow.Task) (*GeneratedCaseMeta, error) {
	caseName := fmt.Sprintf("%s_%s_%s_%s_%s_%s_%s_%s", case2Gen.violateType, case2Gen.identity, case2Gen.plateNo,
		string([]rune(case2Gen.location)[:4]), case2Gen.labels["time"], case2Gen.labels["weather"],
		case2Gen.labels["camAngle"], case2Gen.labels["angle"])

	violateConfig := struct {
		Violations []struct {
			Code string `json:"code"`
		} `json:"violations"`
	}{}
	err := appUtil.ConvByJson(task.AnalyzeConfig, &violateConfig)
	if err != nil || len(violateConfig.Violations) < 1 || violateConfig.Violations[0].Code == "" {
		return nil, errors.New("对应布控任务违法配置错误")
	}
	violateCode, err := strconv.ParseInt(violateConfig.Violations[0].Code, 10, 32)
	if err != nil {
		return nil, errors.New("对应布控任务违法类型配置错误")
	}

	caseMeta := GeneratedCaseMeta{
		Case: &com.MetaCase{
			Name:           caseName,
			Product:        "",
			ProductVersion: "",
			Label: map[string]string{
				"generate":     "auto",
				"id":           case2Gen.violateType + "_" + case2Gen.identity,
				"violate":      case2Gen.violateType,
				"location":     case2Gen.location,
				"origin_video": filepath.Base(case2Gen.OriginVideoPath),
			},
		},
		Task: &TaskMeta{
			AnalyzeConfig: task.AnalyzeConfig,
			Name:          caseName,
			Type:          task.Type,
		},
		Event: []*EventMeta{&EventMeta{
			ClassStr:  []string{case2Gen.vehicleType},
			EventType: int(violateCode),
			Summary: map[string]map[string]interface{}{
				"plate": map[string]interface{}{
					"label": case2Gen.plateNo,
				},
			},
		}},
		Files: &com.FileCase{
			Videos: []string{case2Gen.remoteVideoPath},
			Images: []string{case2Gen.remoteSnapshotPath},
		},
	}
	for label, value := range case2Gen.labels {
		caseMeta.Case.Label[label] = value
	}

	return &caseMeta, nil
}

func getFullAccessPolicy(bucket string) string {
	return strings.Replace(fullAccessPolicyTmpl, "BUCKETNAME", bucket, -1)
}
