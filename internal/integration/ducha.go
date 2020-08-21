package integration

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"git.supremind.info/product/visionmind/com/flow"
	testClient "git.supremind.info/product/visionmind/test/event_test/internal/client"
	"git.supremind.info/product/visionmind/test/event_test/internal/com"
	"git.supremind.info/product/visionmind/util"
	"qiniupkg.com/x/log.v7"
)

type DuchaTester struct {
	TestArgs *TestArgs
	*com.TestCommon
}

func NewDuchaTester(testArgs *TestArgs, testComm *com.TestCommon) *DuchaTester {
	return &DuchaTester{
		TestArgs:   testArgs,
		TestCommon: testComm,
	}
}

func (t *DuchaTester) CreateTask(caseMeta *com.MetaData, imgUrls, videoUrls []string) (err error) {
	if len(imgUrls) < 1 || len(videoUrls) < 1 {
		return errors.New("截图或视频数量小于1")
	}

	var (
		retryTimes = 0
		devices    []*flow.Device
	)

	for i, video := range videoUrls {
		retryTimes = 0
		for retryTimes < 3 {
			var device *flow.Device
			device, err = CreateSubDevice(t.FlowClient, fmt.Sprintf("%s_%d", caseMeta.Case.Name, i), video, t.TestArgs.MaxChannel,
				t.Config.GlobalDeviceID, t.Config.NamePrefix)
			if err == nil {
				devices = append(devices, device)
				break
			}
			retryTimes++
			time.Sleep(2 * time.Second)
		}
		if err != nil {
			return
		}
	}

	retryTimes = 0
	for retryTimes < 3 {
		err = t.createTask(caseMeta.Task, devices, imgUrls)
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

func (t *DuchaTester) createTask(task *flow.Task, devices []*flow.Device, snapshots []string) (err error) {
	if task == nil {
		err = errors.New("task is null")
		log.Errorf("createTask: %+v", err)
		return
	}

	analyzerConfig := struct {
		Cameras []interface{} `json:"cameras"`
	}{}
	err = util.ConvByJson(task.AnalyzeConfig, &analyzerConfig)
	if err != nil {
		return fmt.Errorf("case.json文件格式错误: %s", err.Error())

	}
	if len(devices) != len(analyzerConfig.Cameras) || len(devices) != len(snapshots) {
		return errors.New("analyzer_config.cameras、videos、images 长度不一致")
	}

	parts := strings.SplitN(t.Config.GlobalDeviceID, ".", 2)
	if len(parts) < 2 {
		err = errors.New("invalid vms device")
		return
	}
	task.ID = t.Config.NamePrefix + "_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	task.Name += "_" + t.Config.NamePrefix
	task.AnalyzeConfig["enable_tracking_debug"] = true
	task.Weight = len(devices)
	streamSettings := []flow.StreamSetting{}
	for i, dev := range devices {
		streamId := parts[0] + "." + dev.ID
		streamSettings = append(streamSettings, flow.StreamSetting{
			StreamID: streamId,
			Snapshot: snapshots[i],
		})

		if cam, ok := analyzerConfig.Cameras[i].(map[string]interface{}); ok {
			cam["id"] = streamId
			cam["name"] = dev.Name
		}
	}
	task.StreamList = streamSettings
	task.AnalyzeConfig["cameras"] = analyzerConfig.Cameras

	err = t.TaskClient.CreateTask(task)
	if err != nil {
		log.Errorf("t.TaskClient.CreateTask(%+v):%+v", task, err)
	}

	return
}

func (t *DuchaTester) StartTask(caseMeta *com.MetaData) (err error) {
	err = t.TaskClient.StartTask(caseMeta.Task.ID)
	if err != nil {
		log.Errorf("t.TaskClient.StartTask(%s): %+v", caseMeta.Task.ID, err)
	}

	return
}

func (t *DuchaTester) StopTask(caseMeta *com.MetaData) (err error) {
	err = t.TaskClient.StopTask(caseMeta.Task.ID)
	return
}

func (t *DuchaTester) NewEventClient(caseMeta *com.MetaData) testClient.IEventClient {
	return testClient.NewDuchaEventClient(t.FlowClient)
}

func (t *DuchaTester) Clean(caseMeta *com.MetaData) error {
	//clean task
	err := t.TaskClient.DeleteTask(caseMeta.Task.ID)
	if err != nil {
		log.Errorf("delete task %s failed: %s", caseMeta.Task.ID, err.Error())
	}

	//clean cameras
	streams := []string{}
	for _, stream := range caseMeta.Task.StreamList {
		streams = append(streams, stream.StreamID)
	}
	_, err = t.FlowClient.DeleteDevice(context.Background(), streams)
	if err != nil {
		log.Errorf("delete cameras failed: %s", err.Error())
	}

	return err
}
