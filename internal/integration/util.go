package integration

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"git.supremind.info/product/visionmind/com/flow"
	client "git.supremind.info/product/visionmind/com/go_sdk"
	"git.supremind.info/product/visionmind/test/event_test/internal/com"
	"git.supremind.info/product/visionmind/util"
	"qiniupkg.com/x/log.v7"
)

const VersionKey = "__visionmind_version__"

func ParseCasesFile(path string) (cases []string) {
	cases = []string{}
	filepath.Walk(path,
		func(path string, f os.FileInfo, err error) error {
			if f == nil {
				return err
			}
			if f.IsDir() {
				return nil
			}
			if strings.Contains(path, ".json") {
				cases = append(cases, path)
			}
			return nil
		})

	return
}

func ReadFile(path string) (bytes []byte, err error) {
	file, err := os.Open(path)
	if err != nil {
		log.Errorf("os.Open(%s): %+v", path, err)
		return
	}
	defer file.Close()

	bytes, err = ioutil.ReadAll(file)
	if err != nil {
		log.Errorf("ioutil.ReadAll(%s): %+v", path, err)
		return
	}

	return
}

func GetMetaData(metaDataPath string) (metaData com.MetaData, err error) {
	bytes, err := ReadFile(metaDataPath)
	if err != nil {
		return
	}

	err = json.Unmarshal(bytes, &metaData)
	if err != nil {
		log.Errorf("json.Unmarshal(%+v): %+v", bytes, err)
		return
	}

	return
}

func GetSoftware(flowClient client.IFlowClient) (*Software, error) {
	config, err := flowClient.ConfigCheck(context.Background(), VersionKey)
	if err != nil {
		log.Errorf("flowClient.ConfigCheck(context.Background(), %s): %+v", VersionKey, err)
		return nil, err
	}
	var software Software
	err = util.ConvByJson(config.Info, &software)
	if err != nil {
		log.Errorf("util.ConvByJson(%+v): %+v", config.Info, err)
		return nil, err
	}

	return &software, nil
}

func ConvertReportSlice2Str(reportSlice ReportSlice) (ret string) {
	for index, report := range reportSlice {
		result := "成功"
		if report.ErrorMsg != "" {
			result = "失败"
		}

		id := report.Task.ID
		if id != "" {
			id = "`" + id + "`"
		}
		name := report.Task.Name
		if name != "" {
			name = "`" + name + "`"
		}

		lineStr := fmt.Sprintf("|%d|%s|%s|%s|%s|%s|%s|%s|",
			index+1,
			report.Case.Name,
			report.Case.Product,
			id,
			name,
			report.Task.Type,
			result,
			report.ErrorMsg)

		ret += fmt.Sprintln(lineStr)
	}

	return
}

func ConvertSoftwareImages2Str(detail SoftwareDetail) (ret string) {
	if detail.Details.Images == nil {
		return
	}

	count := 0
	for key, image := range detail.Details.Images {
		count++
		lineStr := fmt.Sprintf("|%d|%s|%s|",
			count,
			key,
			image)

		ret += fmt.Sprintln(lineStr)
	}

	return
}

func GetDurationStr(startTime time.Time) string {
	duration := time.Now().Sub(startTime)

	minute := int(duration.Minutes())
	second := int(duration.Seconds()) - minute*60

	return fmt.Sprintf("%d分%d秒", minute, second)
}

func GerMdPreformatted(s string) string {
	if len(s) > 0 {
		return "`" + s + "`"
	}
	return s
}

func DownloadAsset(url string, dir string, name string) error {
	if !strings.HasPrefix(url, "http") {
		url = "http://" + url
	}

	client := http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Errorf("http.NewRequest(): %s", err.Error())
		return err
	}

	response, err := client.Do(req)
	if err != nil {
		log.Errorf("http.client.Do(): %s", err.Error())
		return err
	}
	defer response.Body.Close()

	newName := filepath.Join(dir, name)
	out, err := os.Create(newName)

	if err != nil {
		log.Errorf("os.Create(): %s", err.Error())
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, response.Body)
	if err != nil {
		log.Errorf("io.Copy(http,%s): %s", newName, err.Error())
	}
	return err
}

func AuthLogin(host, username, password string) (token string, err error) {
	path := "http://" + host + "/v1/auth/login/login"
	requestBody := struct {
		UserName string `json:"username"`
		Password string `json:"password"`
	}{UserName: username, Password: password}
	requestBs, err := json.Marshal(&requestBody)
	if err != nil {
		return
	}

	resp, err := http.Post(path, "application/json", bytes.NewReader(requestBs))
	if err != nil {
		return
	}

	response := struct {
		UserData struct {
			Token string `json:"token"`
		} `json:"userData"`
	}{}

	respBs, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return
	}

	err = json.Unmarshal(respBs, &response)
	if err != nil {
		return
	}

	return response.UserData.Token, nil
}

func CreateSubDevice(flowCli client.IFlowClient, deviceName, url string, maxChan int, globalDevId, namePrefix string) (device *flow.Device, err error) {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(maxChan-1)))
	if err != nil {
		log.Errorf("rand.Int(rand.Reader, big.NewInt(TestArg.MaxChannel)): %+v", err)
		return
	}
	channel := int(n.Int64()) + 1
	subDevice := CreateSubDeviceParams{
		Type:           1,
		OrganizationID: "000000000000000000000000",
		DeviceID:       globalDevId,
		Channel:        channel,
		Attribute: SubDeviceAttribute{
			Name:              namePrefix + "_" + deviceName,
			DiscoveryProtocol: 2,
			UpstreamURL:       url,
			Vendor:            1,
		},
	}

	device, err = flowCli.CreateDevice(context.Background(), subDevice)
	if err != nil {
		log.Errorf("t.FlowClient.CreateDevice(%+v):%+v", subDevice, err)
	}

	return
}
