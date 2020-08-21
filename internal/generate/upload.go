package generate

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"github.com/minio/minio-go/v6"
	"git.supremind.info/product/visionmind/test/event_test/internal/client"
	"git.supremind.info/product/visionmind/util"
)

func (g *Generate) upload(case2Gen *Case, localVideoPath, localSnapshotPath string) (video, snapshot string, err error) {
	taskInfo, ok := g.taskJson[fmt.Sprintf("%s_%s", case2Gen.violateType, case2Gen.location)]
	if !ok {
		err = errors.New("对应布控任务不存在")
		return
	}
	taskType := taskInfo.Type
	taskTypeChn := mapTaskType(taskType)
	if taskType == "" {
		err = fmt.Errorf("unexpected task type %s", taskType)
		return
	}

	video = fmt.Sprintf("交通/%s/%s/%s/%s", taskTypeChn, case2Gen.violateType,
		case2Gen.location, filepath.Base(localVideoPath))
	snapshot = fmt.Sprintf("交通/%s/snapshot/%s", taskTypeChn, filepath.Base(localSnapshotPath))
	if g.arg.RemoteBasePath != "" {
		video = g.arg.RemoteBasePath + "/" + video
		snapshot = g.arg.RemoteBasePath + "/" + snapshot
	}

	err = g.doUpload(video, localVideoPath, "video/mp4")
	if err != nil {
		return
	}

	err = g.doUpload(snapshot, localSnapshotPath, "image/jpeg")
	if err != nil {
		return
	}

	video = fmt.Sprintf("http://%s/%s/%s", g.config.Minio.Host, g.config.Minio.Bucket, video)
	snapshot = fmt.Sprintf("http://%s/%s/%s", g.config.Minio.Host, g.config.Minio.Bucket, snapshot)
	return
}

func (g *Generate) doUpload(remotePath, localPath, contentType string) (err error) {
	putOpts := minio.PutObjectOptions{ContentType: contentType}
	_, err = g.s3Client.StatObject(g.config.Minio.Bucket, remotePath, minio.StatObjectOptions{})
	if err != nil && minio.ToErrorResponse(err).Code != "NoSuchKey" {
		return
	}

	if err != nil || g.arg.OverWrite {
		_, err = g.s3Client.FPutObject(g.config.Minio.Bucket, remotePath, localPath, putOpts)
	}
	return
}

func mapTaskType(taskType string) string {
	switch taskType {
	case "jdc":
		return "机动车"
	case "fjdc":
		return "非机动车"
	default:
		return ""
	}
}

func (g *Generate) ensureForkExists() (string, error) {
	splitedRepoPath := strings.Split(g.config.Git.Repo, "/")
	forkedRepo := g.config.Git.UserName + "/" + splitedRepoPath[len(splitedRepoPath)-1]
	repo, err := g.gitlabCli.GetProject(context.Background(), forkedRepo)
	if err != nil {
		if err != client.ErrNotFound {
			return "", err
		}
		return g.gitlabCli.ForkFrom(context.Background(), g.config.Git.Repo)
	}

	if repo.ForkedFrom.Name == "" || repo.ForkedFrom.PathWithNamespace != g.config.Git.Repo {
		return "", errors.New(forkedRepo + " is not forked from " + g.config.Git.Repo)
	}
	return forkedRepo, nil
}

func (g *Generate) pushNewCases(remoteRepo, newBranch, casePath, gitBasePath, commitMsg string) (err error) {
	var (
		gitTmpDir     = filepath.Join(g.arg.BasePath, "git_tmp")
		gitRemote     = fmt.Sprintf("%s/%s.git", g.config.Git.GitLab, remoteRepo)
		gitTargetPath = filepath.Join(gitTmpDir, gitBasePath)
	)

	repo, workTree, err := g.gitlabCli.CloneAndCheckout(gitRemote, newBranch, gitTmpDir)
	if err != nil {
		return err
	}

	//copy cases into git stage space
	os.MkdirAll(gitTargetPath, 0755)
	err = util.CopyRecursively(casePath, gitTargetPath)
	if err != nil {
		return err
	}

	err = g.gitlabCli.AddAndPush(repo, workTree, gitBasePath, commitMsg)
	return err
}
