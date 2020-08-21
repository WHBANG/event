package generate

import (
	"git.supremind.info/product/visionmind/test/event_test/internal/com"
)

type GeneratedCaseMeta struct {
	Case  *com.MetaCase `json:"case"`
	Task  *TaskMeta     `json:"task"`
	Event []*EventMeta  `json:"event"`
	Files *com.FileCase `json:"files"`
}

type TaskMeta struct {
	AnalyzeConfig interface{} `json:"analyze_config"`
	Name          string      `json:"name"`
	Type          string      `json:"type"`
}

type EventMeta struct {
	ClassStr  []string                          `json:"classStr"`
	EventType int                               `json:"eventType"`
	Summary   map[string]map[string]interface{} `json:"summary"`
}

type GenerateArg struct {
	Verbose        bool
	BasePath       string
	OverWrite      bool
	Upload         bool
	CutOnly        bool
	Args           string
	RemoveBFrame   bool
	RemoteBasePath string
	PR             bool
}

type Case struct {
	identity           string //case标识，测试集内唯一
	csvName            string
	csvIndex           int
	violateType        string
	OriginVideoPath    string
	location           string
	vehicleType        string
	plateNo            string
	startTime          string
	endTime            string
	remoteVideoPath    string
	remoteSnapshotPath string
	labels             map[string]string
}
