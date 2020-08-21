module git.supremind.info/product/visionmind/test/event_test

go 1.13

require (
	git.supremind.info/product/app/traffic/com v0.0.0-20200811075917-7dfed71d1d95
	git.supremind.info/product/visionmind/com v0.0.0-20200807084330-cff77eb16e5d
	git.supremind.info/product/visionmind/sdk/vmr/go_sdk v0.0.0-20200806092421-705c9de5ba5f
	git.supremind.info/product/visionmind/util v0.0.0-20200820004331-3da87d4df109
	github.com/go-git/go-git/v5 v5.1.0
	github.com/imdario/mergo v0.3.10
	github.com/klauspost/cpuid v1.3.1 // indirect
	github.com/minio/minio-go/v6 v6.0.57
	github.com/spf13/cobra v1.0.0
	golang.org/x/crypto v0.0.0-20200709230013-948cd5f35899 // indirect
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e
	gopkg.in/ini.v1 v1.57.0 // indirect
	qbox.us v0.0.0-00010101000000-000000000000
	qiniupkg.com/x v7.0.0+incompatible
)

replace (
	github.com/qiniu => git.supremind.info/product/visionmind/com/dep/github.com/qiniu v0.0.0-20200807084330-cff77eb16e5d
	github.com/qiniu/db/mgoutil.v3 => git.supremind.info/product/visionmind/com/dep/github.com/qiniu/db/mgoutil.v3 v0.0.0-20200807084330-cff77eb16e5d
	qbox.us => git.supremind.info/product/visionmind/com/dep/qbox.us v0.0.0-20200807084330-cff77eb16e5d
	qiniu.com => git.supremind.info/product/visionmind/com/dep/qiniu.com v0.0.0-20200807084330-cff77eb16e5d
	qiniupkg.com => git.supremind.info/product/visionmind/com/dep/qiniupkg.com v0.0.0-20200807084330-cff77eb16e5d
	qiniupkg.com/x => git.supremind.info/product/visionmind/com/dep/qiniupkg.com/x v0.0.0-20200807084330-cff77eb16e5d
	qiniupkg.com/x/rollog.v1 => git.supremind.info/product/visionmind/com/dep/qiniupkg.com/x/rollog.v1 v0.0.0-20200807084330-cff77eb16e5d
)
