package integration

const reportTemplate = `# 集成测试

## 服务版本简介

* 测试版本-APP: {{ .Soft.Software.App.Surveillance.Version }}
* 测试版本-VMR: {{ .Soft.Software.Platform.Vmr.Version }}
* 测试开始时间: {{ .StartTime.Format "2006-01-02 15:04:05" }}
* 测试运行时间: {{ GetDurationStr .StartTime }}

## Case 结果列表

| 测试编号 | 测试名称 | 测试产品 | 任务ID | 任务名称 | 任务类型 | 测试结果 | 失败原因 |
| :------| :------| :------| :------| :------| :------| :------| :------|
{{- range $idx, $report := .ReportSlice }}
| {{ $idx }} | {{ $report.Case.Name }} | {{ $report.Case.Product }} | {{ GerMdPreformatted $report.Task.ID }} | {{ GerMdPreformatted $report.Task.Name }} | {{ $report.Task.Type }} | {{ if $report.ErrorMsg }} 失败 {{ else }} 成功 {{ end }} | {{ GerMdPreformatted $report.ErrorMsg }} |
{{- end }}

## 服务版本的详细信息

### APP 服务版本

| 服务名称 | 镜像版本 |
| :------| :------|
{{- range $name, $image := .Soft.Software.App.Surveillance.Details.Images }}
| {{ GerMdPreformatted $name }} | {{ GerMdPreformatted $image }} |
{{- end }}


### VMR 服务版本

| 服务名称 | 镜像版本 |
| :------| :------|
{{- range $name, $image := .Soft.Software.Platform.Vmr.Details.Images }}
| {{ GerMdPreformatted $name }} | {{ GerMdPreformatted $image }} |
{{- end }}
`
