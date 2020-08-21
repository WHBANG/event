package generate

const FailedTemplate = `
## 测试用例生成
### 创建时间: {{ .StartAt.Format "2006-01-02 15:04:05" }}
### 失败数/总数: {{ .FailedNum }}/{{ .CaseNum }}

{{ range $csvName,$cases:= .Cases }}
### {{ $csvName }}
{{ range $idx,$case:= $cases }}
| 出错行号 | 错误消息 |
| :------| :------|
|{{ $case.CsvIndex }} |{{ $case.Msg }} |
{{ end }}

{{ end }}
`

const fullAccessPolicyTmpl = `
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Principal": {
                "AWS": [
                    "*"
                ]
            },
            "Action": [
                "s3:GetBucketLocation",
                "s3:ListBucketMultipartUploads"
            ],
            "Resource": [
                "arn:aws:s3:::BUCKETNAME"
            ]
        },
        {
            "Effect": "Allow",
            "Principal": {
                "AWS": [
                    "*"
                ]
            },
            "Action": [
                "s3:ListBucket"
            ],
            "Resource": [
                "arn:aws:s3:::BUCKETNAME"
            ],
            "Condition": {
                "StringEquals": {
                    "s3:prefix": [
                        "*.*"
                    ]
                }
            }
        },
        {
            "Effect": "Allow",
            "Principal": {
                "AWS": [
                    "*"
                ]
            },
            "Action": [
                "s3:AbortMultipartUpload",
                "s3:DeleteObject",
                "s3:GetObject",
                "s3:ListMultipartUploadParts",
                "s3:PutObject"
            ],
            "Resource": [
                "arn:aws:s3:::BUCKETNAME/*.**"
            ]
        }
    ]
}
`
