package integration

import (
	"context"
	"crypto/rand"
	"errors"
	"math/big"
	"strconv"
	"strings"
	"time"

	"git.supremind.info/product/visionmind/com/flow"
	testClient "git.supremind.info/product/visionmind/test/event_test/internal/client"
	"git.supremind.info/product/visionmind/test/event_test/internal/com"
	"qiniupkg.com/x/log.v7"
)

type JiaotongTester struct {
	TestArgs *TestArgs
	*com.TestCommon
}

func NewJiaotongTester(testArgs *TestArgs, testComm *com.TestCommon) *JiaotongTester {
	return &JiaotongTester{
		TestArgs:   testArgs,
		TestCommon: testComm,
	}
}

func (t *JiaotongTester) CreateTask(caseMeta *com.MetaData, imgUrls, videoUrls []string) (err error) {
	if len(imgUrls) < 1 || len(videoUrls) < 1 {
		return errors.New("截图或视频数量小于1")
	}
	var (
		retryTimes = 0
		device     *flow.Device
	)
	for retryTimes < 3 {
		device, err = CreateSubDevice(t.FlowClient, caseMeta.Case.Name, videoUrls[0], t.TestArgs.MaxChannel,
			t.Config.GlobalDeviceID, t.Config.NamePrefix)
		if err == nil {
			break
		}
		retryTimes++
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		return
	}

	retryTimes = 0
	for retryTimes < 3 {
		err = t.createTask(caseMeta.Task, device.ID, imgUrls[0])
		if err == nil {
			break
		}
		retryTimes++
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		return
	}

	return
}

func (t *JiaotongTester) StartTask(caseMeta *com.MetaData) (err error) {
	err = t.TaskClient.StartTask(caseMeta.Task.ID)
	if err != nil {
		log.Errorf("t.TaskClient.StartTask(%s): %+v", caseMeta.Task.ID, err)
	}

	return
}

func (t *JiaotongTester) StopTask(caseMeta *com.MetaData) (err error) {
	err = t.TaskClient.StopTask(caseMeta.Task.ID)
	return
}

// 创建子设备
func (t *JiaotongTester) createSubDevice(deviceName, url string) (device *flow.Device, err error) {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(t.TestArgs.MaxChannel-1)))
	if err != nil {
		log.Errorf("rand.Int(rand.Reader, big.NewInt(TestArg.MaxChannel)): %+v", err)
		return
	}
	channel := int(n.Int64()) + 1
	subDevice := CreateSubDeviceParams{
		Type:           1,
		OrganizationID: "000000000000000000000000",
		DeviceID:       t.Config.GlobalDeviceID,
		Channel:        channel,
		Attribute: SubDeviceAttribute{
			Name:              t.Config.NamePrefix + "_" + deviceName,
			DiscoveryProtocol: 2,
			UpstreamURL:       url,
			Vendor:            1,
		},
	}

	device, err = t.FlowClient.CreateDevice(context.Background(), subDevice)
	if err != nil {
		log.Errorf("t.FlowClient.CreateDevice(%+v):%+v", subDevice, err)
	}

	return
}

// 创建任务
func (t *JiaotongTester) createTask(task *flow.Task, streamID, snapshotURL string) (err error) {
	if task == nil {
		err = errors.New("task is null")
		log.Errorf("createTask: %+v", err)
		return
	}

	parts := strings.SplitN(t.Config.GlobalDeviceID, ".", 2)
	if len(parts) < 2 {
		err = errors.New("invalid vms device")
		return
	}
	streamID = parts[0] + "." + streamID

	task.ID = t.Config.NamePrefix + "_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	task.Name += "_" + t.Config.NamePrefix
	task.StreamID = streamID
	task.Snapshot = snapshotURL
	task.StreamON = "ON"
	task.AnalyzeConfig["enable_tracking_debug"] = true
	task.Region = t.TestArgs.Region
	violations, ok := task.AnalyzeConfig["violations"].([]interface{})
	if !ok {
		return errors.New("case.json文件格式错误")
	}
	for i, _ := range violations {
		if v, ok := violations[i].(map[string]interface{}); ok {
			v["on"] = true
		}
	}

	task.AnalyzeConfig["tracking_threshold"] = 0.6

	err = t.TaskClient.CreateTask(task)
	if err != nil {
		log.Errorf("t.TaskClient.CreateTask(%+v):%+v", task, err)
	}

	return
}

func (t *JiaotongTester) NewEventClient(caseMeta *com.MetaData) testClient.IEventClient {
	return testClient.NewJtEventClient(t.Config.ConsoleHost, nil)
}

func (t *JiaotongTester) Clean(caseMeta *com.MetaData) error {
	//clean task
	err := t.TaskClient.DeleteTask(caseMeta.Task.ID)
	if err != nil {
		log.Errorf("delete task %s failed: %s", caseMeta.Task.ID, err.Error())
	}

	//clean camera
	_, err = t.FlowClient.DeleteDevice(context.Background(), []string{caseMeta.Task.StreamID})
	if err != nil {
		log.Errorf("delete camera failed: %s", err.Error())
	}

	return err
}
