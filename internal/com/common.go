package com

import (
	"encoding/json"

	"git.supremind.info/product/visionmind/com/flow"
	"git.supremind.info/product/visionmind/com/go_sdk"
	"git.supremind.info/product/visionmind/sdk/vmr/go_sdk"
	"github.com/imdario/mergo"
	"qbox.us/cc/config"
	"qiniupkg.com/x/log.v7"
)

const DefaultWorker int = 5

type Config struct {
	FlowHost       string `json:"flow_host"`
	FileserverHost string `json:"fileserver_host"`
	ConsoleHost    string `json:"console_host"`
	GlobalDeviceID string `json:"global_device_id"`
	NamePrefix     string `json:"name_prefix"`
	EventTimeout   int    `json:"event_timeout"`  //分钟为单位
	EventDuration  int    `json:"event_duration"` //秒为单位
	CasesVolume    string `json:"cases_volume"`
	CasesPrefix    string `json:"cases_prefix"`
	Worker         int    `json:"worker"`
	AuthHost       string `json:"auth_host"`
	AuthUserName   string `json:"auth_username"`
	AuthPassword   string `json:"auth_password"`
	Minio          struct {
		Host            string `json:"host"`
		AccessKeyID     string `json:"accesskey"`
		SecretAccessKey string `json:"secret_accesskey"`
		UseSSL          bool   `json:"use_ssl"`
		Bucket          string `json:"bucket"`
	} `json:"minio"`
	Git struct {
		GitLab      string `json:"gitlab"`
		Repo        string `json:"repo"`
		UserName    string `json:"username"`
		AccessToken string `json:"access_token"`
	} `json:"git"`
}

var defaultConfig = Config{
	EventTimeout:  5,
	EventDuration: 30,
}

func ParseConfigFile(configPath string) (conf *Config) {
	conf = &Config{}
	if e := config.LoadEx(conf, configPath); e != nil {
		log.Fatal("config.Load failed:", e)
	}

	mergo.Merge(conf, &defaultConfig)
	buf, _ := json.MarshalIndent(conf, "", "    ")
	log.Infof("loaded conf \n%s", string(buf))

	return
}

type TestCommon struct {
	Config           Config
	FlowClient       go_sdk.IFlowClient
	TaskClient       go_sdk.ITaskMgmClient
	FileserverClient go_sdk.IFileServerClient
}

func NewTestCommon(configPath string) (test *TestCommon, err error) {
	conf := ParseConfigFile(configPath)

	flowClient := client.NewFlowClient(conf.FlowHost)
	taskClient := client.NewTaskMgmClient(conf.FlowHost)
	fileserverClient := client.NewFileServerClient(conf.FileserverHost)
	err = fileserverClient.Init()
	if err != nil {
		log.Fatal("failed to init the fileserver client", err)
	}

	if conf.NamePrefix == "" {
		conf.NamePrefix = "test"
	}
	if conf.Worker <= 0 {
		conf.Worker = DefaultWorker
	}

	test = &TestCommon{
		Config:           *conf,
		FlowClient:       flowClient,
		TaskClient:       taskClient,
		FileserverClient: fileserverClient,
	}

	return
}

type EventData map[string]interface{}

type MetaData struct {
	Case  *MetaCase   `json:"case"`
	Task  *flow.Task  `json:"task"`
	Event []EventData `json:"event"`
	Files *FileCase   `json:"files"`
}

type FileCase struct {
	Videos []string `json:"videos"`
	Images []string `json:"images"`
}

type MetaCase struct {
	Name           string            `json:"name"`
	Product        string            `json:"product"`
	ProductVersion string            `json:"product_version"`
	Label          map[string]string `json:"label"`
}
