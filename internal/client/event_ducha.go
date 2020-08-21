package client

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	client "git.supremind.info/product/visionmind/com/go_sdk"
	"git.supremind.info/product/visionmind/test/event_test/internal/com"
	"git.supremind.info/product/visionmind/util"
	"qiniupkg.com/x/log.v7"
)

type DuchaEventClient struct {
	flowCli client.IFlowClient
	since   time.Time
}

func NewDuchaEventClient(flowCli client.IFlowClient) *DuchaEventClient {
	return &DuchaEventClient{
		flowCli: flowCli,
		since:   time.Now(),
	}
}

func (c *DuchaEventClient) NextEvents(ctx context.Context, caseMeta *com.MetaData, dur time.Duration, token string) ([]com.EventData, error) {
	events := []com.EventData{}
	beginTime := time.Now()

	topic := caseMeta.Task.Type + "_" + "alarm"
	for {
		select {
		case <-ctx.Done():
			return events, ErrCanceled
		default:
		}
		var (
			event     = DuchaEvent{}
			eventData = map[string]interface{}{}
			ok        bool
		)
		msg, err := c.flowCli.PullSince(ctx, topic, caseMeta.Task.ID, c.since)
		if err != nil {
			if !strings.Contains(err.Error(), "timeout") {
				log.Errorf("pull event: %s", err.Error())
			}
			goto CHECK_TIMEOUT
		}
		eventData, ok = msg.Body.(map[string]interface{})
		err = util.ConvByJson(msg.Body, &event)
		if !ok || err != nil {
			log.Errorf("failed parsing event,drop it: %+v", err)
			goto CHECK_TIMEOUT
		}

		if event.TaskId == caseMeta.Task.ID {
			events = append(events, eventData)
		}
	CHECK_TIMEOUT:
		if len(events) > 0 && time.Now().Sub(beginTime) >= dur {
			return events, nil
		}
	}
}

type DuchaEvent struct {
	TaskId  string `json:"task_id"`
	ModelId string `json:"model_id"`
	SceneId string `json:"scene_id"`
}

func (c *DuchaEventClient) Match(caseEvent *com.EventData, msgs []com.EventData) error {
	var err error = nil
	eventExpect := DuchaEvent{}
	err = util.ConvByJson(caseEvent, &eventExpect)
	if err != nil {
		return err
	}

	for _, msg := range msgs {
		errStr := ""
		fetchedEvent := DuchaEvent{}
		err = util.ConvByJson(msg, &fetchedEvent)
		if err != nil {
			return err
		}

		if eventExpect.ModelId != fetchedEvent.ModelId {
			errStr += fmt.Sprintf("发现不匹配的model_id,期望值%s,实际值%s ", eventExpect.ModelId, fetchedEvent.ModelId)
		}

		if eventExpect.SceneId != fetchedEvent.SceneId {
			errStr += fmt.Sprintf("发现不匹配的scene_id,期望值%s,实际值%s ", eventExpect.SceneId, fetchedEvent.SceneId)
		}

		if errStr == "" {
			return nil
		} else {
			err = errors.New(errStr)
		}
	}
	return err
}
