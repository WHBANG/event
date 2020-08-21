package clear

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"path"
	"strings"

	"git.supremind.info/product/visionmind/com/flow"
	"git.supremind.info/product/visionmind/test/event_test/internal/com"
	"qiniupkg.com/x/log.v7"
)

type Clear struct {
	com.TestCommon
	TaskType string
}

func (c *Clear) Do() (err error) {
	huanhang := "\r\n"
	var content string

	successList, err := c.clearTasks()
	if err == nil {
		if len(successList) > 0 {
			for index, id := range successList {
				if index == 0 {
					content += "清空任务IDs:" + huanhang
				}
				content += id + huanhang
			}
		} else {
			log.Infof("no tasks to clear")
		}
	}

	successList, err = c.clearDevices()
	if err == nil {
		if len(successList) > 0 {
			for index, id := range successList {
				if index == 0 {
					content += "清空设备IDs:" + huanhang
				}
				content += id + huanhang
			}
		} else {
			log.Infof("no devices to clear")
		}
	}

	if content != "" {
		bytes := []byte(content)

		fileName := path.Join("./success.txt")
		err = ioutil.WriteFile(fileName, bytes, 0755)
		if err != nil {
			err = fmt.Errorf("Do: ioutil.WriteFile(%s): %v", fileName, err)
			log.Errorf("%+v", err)
			return
		}

		log.Info("success ids has been written in success.txt")
	}

	log.Info("clear done")
	return
}

func (c *Clear) clearTasks() (successList []string, err error) {
	req := flow.SearchReqeust{
		Name:   c.Config.NamePrefix,
		Simple: true,
		Type:   c.TaskType,
	}
	tasks, err := c.TaskClient.SearchTask(req)
	if err != nil {
		log.Errorf("c.TaskClient.SearchTask(%+v): %+v", req, err)
		return
	}

	for _, task := range tasks {
		if !strings.HasPrefix(task.ID, c.Config.NamePrefix) {
			continue
		}

		err = c.TaskClient.DeleteTask(task.ID)
		if err != nil {
			log.Errorf("c.TaskClient.DeleteTask(%s): %+v", task.ID, err)
			continue
		}

		successList = append(successList, task.ID)
	}

	return
}

func (c *Clear) clearDevices() (successList []string, err error) {
	devices, err := c.FlowClient.GetDevices(context.Background())
	if err != nil {
		log.Errorf("c.FlowClient.GetDevices(context.Background()): %+v", err)
		return
	}

	parts := strings.SplitN(c.Config.GlobalDeviceID, ".", 2)
	if len(parts) < 2 {
		err = errors.New("invalid vms device")
		log.Errorf("clearDevices(): %+v", err)
		return
	}

	deviceIDs := []string{}
	for _, device := range devices {
		if !strings.HasPrefix(device.Name, c.Config.NamePrefix) {
			continue
		}
		if !strings.HasPrefix(device.ID, parts[0]) {
			continue
		}

		deviceIDs = append(deviceIDs, device.ID)
	}

	if len(deviceIDs) == 0 {
		return
	}

	successList, err = c.FlowClient.DeleteDevice(context.Background(), deviceIDs)
	if err != nil {
		log.Errorf("c.FlowClient.DeleteDevice(context.Background(), %+v): %+v", deviceIDs, err)
		return
	}

	return
}
