package integration

import (
	"git.supremind.info/product/visionmind/test/event_test/internal/com"
)

type CreateSubDeviceParams struct {
	Type           int                `json:"type"`
	OrganizationID string             `json:"organization_id"`
	DeviceID       string             `json:"device_id"`
	Channel        int                `json:"channel"`
	Attribute      SubDeviceAttribute `json:"attribute"`
}

type SubDeviceAttribute struct {
	Name              string `json:"name"`
	DiscoveryProtocol int    `json:"discovery_protocol"`
	UpstreamURL       string `json:"upstream_url"`
	Vendor            int    `json:"vendor"`
}

type Report struct {
	com.MetaData
	ErrorMsg string
}

type ReportSlice []Report

func (c ReportSlice) Len() int {
	return len(c)
}
func (c ReportSlice) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}
func (c ReportSlice) Less(i, j int) bool {
	str1 := c[i].Case.Product + c[i].Task.Type + c[i].Case.Name
	str2 := c[j].Case.Product + c[j].Task.Type + c[j].Case.Name
	return str1 < str2
}

type Software struct {
	Software struct {
		App struct {
			Surveillance SoftwareDetail `json:"surveillance"`
		} `json:"app"`
		Platform struct {
			Vmr SoftwareDetail `json:"vmr"`
			Ca  SoftwareDetail `json:"ca"`
		}
	} `json:"software"`
}

type SoftwareDetail struct {
	Details struct {
		Images map[string]string `json:"images"`
	} `json:"details"`
	Version string `json:"version"`
}
