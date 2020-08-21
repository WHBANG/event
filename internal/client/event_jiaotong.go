package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	proto "git.supremind.info/product/app/traffic/com/proto/console"
	"git.supremind.info/product/visionmind/test/event_test/internal/com"
	"git.supremind.info/product/visionmind/util"
	"qiniupkg.com/x/log.v7"
)

var ErrCanceled = errors.New("GetEvent Canceled")
var ErrUnAuthed = errors.New("Authorization failed")

type IEventClient interface {
	NextEvents(ctx context.Context, caseMeta *com.MetaData, dur time.Duration, token string) ([]com.EventData, error)
	Match(caseEvent *com.EventData, msg []com.EventData) error
}

type JtEventClient struct {
	host string

	marker string
	events []int
}

func NewJtEventClient(host string, events []int) *JtEventClient {
	return &JtEventClient{host: host, events: events}
}

func (client *JtEventClient) GetEvents(taskId string, token string) ([]com.EventData, error) {
	curPage := 1
	totalPage := curPage + 1
	ending := time.Now().UnixNano() / int64(time.Millisecond)
	var results []com.EventData
	for ; curPage < totalPage; curPage++ {
		var builder strings.Builder
		fmt.Fprintf(&builder, "http://%s/v1/events?taskId=%s&page=%d&end=%d&per_page=10", client.host, taskId, curPage, ending)
		if client.marker != "" && curPage == 1 {
			builder.WriteString("&marker=")
			builder.WriteString(client.marker)
		}
		for _, event := range client.events {
			builder.WriteString("&eventTypes=")
			builder.WriteString(strconv.FormatInt(int64(event), 10))
		}
		req, err := http.NewRequest("GET", builder.String(), nil)
		if err != nil {
			return nil, err
		}
		if token != "" {
			req.Header.Set("Authorization", token)
		}

		var subResults proto.CommonRes
		subResults.Data = &proto.PageData{Content: &[]com.EventData{}}

		rep, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		if rep.StatusCode == http.StatusUnauthorized {
			return nil, ErrUnAuthed
		}

		eventsJson, err := ioutil.ReadAll(rep.Body)
		if err != nil {
			return nil, err
		}
		rep.Body.Close()
		err = json.Unmarshal(eventsJson, &subResults)
		if err != nil {
			log.Error("failed to parse response from /v1/events")
			return nil, err
		}
		pageData, ok := subResults.Data.(*proto.PageData)
		if !ok {
			log.Errorf("/v1/events: unexpected response %s", string(eventsJson))
			return results, nil
		}
		eventData, ok := pageData.Content.(*[]com.EventData)
		if !ok {
			log.Errorf("/v1/events: unexpected response %s", string(eventsJson))
			return results, nil
		}
		results = append(results, *eventData...)

		totalPage = pageData.TotalPage + 1
	}
	return results, nil
}

func (client *JtEventClient) NextEvents(ctx context.Context, caseMeta *com.MetaData, dur time.Duration, token string) ([]com.EventData, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ErrCanceled
		default:
		}
		results, err := client.GetEvents(caseMeta.Task.ID, token)
		if err != nil {
			return nil, err
		}
		if len(results) > 0 {
			eventResult := proto.GetEventsData{}
			err = util.ConvByJson(results[0], &eventResult)
			if err != nil {
				return nil, err
			}
			client.marker = eventResult.ID.Hex()
			return results, nil
		} else {
			time.Sleep(dur)
		}
	}
}

func (client *JtEventClient) Match(caseEvent *com.EventData, msg []com.EventData) error {
	var err error = nil

	for i, _ := range msg {
		errStr := ""
		var eventGet, event proto.GetEventsData
		err = util.ConvByJson(caseEvent, &event)
		if err != nil {
			return err
		}
		err = util.ConvByJson(msg[i], &eventGet)
		if err != nil {
			return err
		}

		if event.EventType != eventGet.EventType {
			errStr += fmt.Sprintf("事件类型不匹配,期望值%d,实际值%d ", event.EventType, eventGet.EventType)
		}
		for _, expectClass := range event.ClassStr {
			for _, getClass := range eventGet.ClassStr {
				if expectClass == "" || expectClass == getClass {
					goto HIT
				}
			}
			errStr += fmt.Sprintf("未发现匹配类别，期望值%+v ", expectClass)
		HIT:
		}
		for k, _ := range event.Summary {
			if event.Summary[k].Label != "" && event.Summary[k].Label != eventGet.Summary[k].Label {
				errStr += fmt.Sprintf("%s label不匹配,期望值%s,实际值%s ", k, event.Summary[k].Label, eventGet.Summary[k].Label)
			}
			if event.Summary[k].Score != 0 && math.Abs(event.Summary[k].Score-eventGet.Summary[k].Score) > 0.1 {
				errStr += fmt.Sprintf("%s score值与预期偏差过大，期望值%.3f,实际值%.3f ", k, event.Summary[k].Score, eventGet.Summary[k].Score)
			}
		}
		if errStr == "" {
			return nil
		} else {
			err = errors.New(errStr)
		}
	}
	return err
}
