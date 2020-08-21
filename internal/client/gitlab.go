package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	gitHttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
	"qiniupkg.com/x/log.v7"
	"git.supremind.info/product/visionmind/test/event_test/internal/com"
)

var (
	ErrNotFound = errors.New("404 Not Found")
)

type GitLabErrorResp struct {
	Message string `json:"message"`
}

type GitLabRepoResp struct {
	GitLabErrorResp
	ID         int    `json:"id"`
	Name       string `json:"name"`
	ForkedFrom struct {
		Name              string `json:"name"`
		PathWithNamespace string `json:"path_with_namespace"`
	} `json:"forked_from_project"`
	ImportStatus string      `json:"import_status"`
	ImportError  interface{} `json:"import_error"`
}

type GitLabMRResp struct {
	GitLabErrorResp
	WebUrl string `json:"web_url"`
}

type GitlabClient struct {
	username    string
	accessToken string
	gitlabHost  string
}

func NewGitLabClient(config *com.Config) *GitlabClient {
	return &GitlabClient{
		username:    config.Git.UserName,
		accessToken: config.Git.AccessToken,
		gitlabHost:  config.Git.GitLab,
	}
}

func (g *GitlabClient) GetProject(ctx context.Context, repoPath string) (*GitLabRepoResp, error) {
	reqUrl := fmt.Sprintf("%s/api/v4/projects/%s?private_token=%s", g.gitlabHost, url.PathEscape(repoPath), g.accessToken)
	resp := GitLabRepoResp{}
	err := doJson(ctx, reqUrl, http.MethodGet, nil, &resp)
	if err != nil {
		return nil, err
	}

	if resp.Message != "" {
		return nil, errors.New(resp.Message)
	}
	return &resp, nil
}

func (g *GitlabClient) ForkFrom(ctx context.Context, originRepo string) (string, error) {
	reqUrl := fmt.Sprintf("%s/api/v4/projects/%s/fork?private_token=%s", g.gitlabHost, url.PathEscape(originRepo), g.accessToken)
	err := doJson(ctx, reqUrl, http.MethodPost, nil, nil)
	if err != nil {
		return "", err
	}

	splitedRepoPath := strings.Split(originRepo, "/")
	forkedRepo := g.username + "/" + splitedRepoPath[len(splitedRepoPath)-1]

	//wait for fork complete
	for {
		time.Sleep(2 * time.Second)
		repo, err := g.GetProject(ctx, forkedRepo)
		if err != nil {
			return "", err
		}

		if repo.ImportStatus == "finished" {
			return forkedRepo, nil
		}
		if repo.ImportError != nil {
			return "", errors.New("failed fork repo " + originRepo)
		}
	}
}

func (g *GitlabClient) CreatePR(ctx context.Context, originRepo, forkedRepo, sourceBranch, targetBranch, title string) (string, error) {
	originRepoInfo, err := g.GetProject(ctx, originRepo)
	if err != nil {
		return "", err
	}

	reqUrl := fmt.Sprintf("%s/api/v4/projects/%s/merge_requests?private_token=%s&source_branch=%s&target_branch=%s&title=%s&target_project_id=%d",
		g.gitlabHost, url.PathEscape(forkedRepo), g.accessToken, sourceBranch, targetBranch, url.PathEscape(title), originRepoInfo.ID)
	resp := GitLabMRResp{}
	err = doJson(ctx, reqUrl, http.MethodPost, nil, &resp)
	if err != nil {
		return "", err
	}
	if resp.Message != "" {
		return "", errors.New(resp.Message)
	}

	return resp.WebUrl, nil
}

func doJson(ctx context.Context, url string, method string, reqBody interface{}, respBody interface{}) (err error) {
	var requstBody []byte = nil
	if reqBody != nil {
		requstBody, err = json.Marshal(reqBody)
		if err != nil {
			log.Errorf("failed marshal request body for request %s: %s", url, err.Error())
			return
		}
	}
	req, err := http.NewRequest(method, url, bytes.NewReader(requstBody))
	if err != nil {
		log.Errorf("failed create request for url %s: %s", url, err.Error())
		return
	}

	req = req.WithContext(ctx)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Errorf("failed requesting url %s: %s", url, err.Error())
		return
	}

	if resp.StatusCode/100 != 2 {
		if resp.StatusCode == http.StatusNotFound {
			return ErrNotFound
		}
		if resp.StatusCode == http.StatusUnauthorized {
			return ErrUnAuthed
		}
		err = fmt.Errorf("unexpected status code %d for url %s", resp.StatusCode, url)
		log.Error(err.Error())
		return
	}

	defer resp.Body.Close()
	if respBody != nil {
		respBodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Errorf("failed reading response body of request %s: %s", url, err.Error())
			return err
		}
		err = json.Unmarshal(respBodyBytes, respBody)
		if err != nil {
			err = fmt.Errorf("failed unmarshal response into type %T for url %s: %s", respBody, url, err.Error())
		}
	}

	return
}

func (g *GitlabClient) CloneAndCheckout(gitRemote, newBranch, gitTmpPath string) (repo *git.Repository, workTree *git.Worktree, err error) {
	os.RemoveAll(gitTmpPath)

	repo, err = git.PlainClone(gitTmpPath, false, &git.CloneOptions{
		URL: gitRemote,
		Auth: &gitHttp.BasicAuth{
			Username: g.username,
			Password: g.accessToken,
		},
		Depth: 1,
	})
	if err != nil {
		log.Errorf("failed clone git repo from %s: %+v", gitRemote, err)
		return nil, nil, err
	}

	head, err := repo.Head()
	if err != nil {
		log.Errorf("failed get HEAD in git repo: %+v", err)
		return nil, nil, err
	}
	ref := plumbing.NewHashReference(plumbing.ReferenceName("refs/heads/"+newBranch), head.Hash())
	err = repo.Storer.SetReference(ref)
	if err != nil {
		log.Errorf("failed create new branch %s: %+v", newBranch, err)
		return nil, nil, err
	}

	workTree, err = repo.Worktree()
	if err != nil {
		log.Errorf("failed getting worktree from git_tmp: %+v", err)
		return nil, nil, err
	}

	err = workTree.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(newBranch),
		Force:  true,
	})
	if err != nil {
		log.Errorf("failed switch to branch %s: %+v", newBranch, err)
		return nil, nil, err
	}
	log.Infof("switched to new branch %s", newBranch)
	return
}

func (g *GitlabClient) AddAndPush(repo *git.Repository, workTree *git.Worktree, casePath, commitMsg string) error {
	_, err := workTree.Add(casePath)
	if err != nil {
		log.Errorf("failed adding %s into workspace: %+v", casePath, err)
		return err
	}

	commitHash, err := workTree.Commit(commitMsg, &git.CommitOptions{
		Author: &object.Signature{
			Name:  g.username,
			Email: g.username + "@supremind.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		log.Errorf("failed commit change: %+v", err)
		return err
	}

	err = repo.Push(&git.PushOptions{
		RemoteName: "origin",
		Progress:   os.Stdout,
		Auth: &gitHttp.BasicAuth{
			Username: g.username,
			Password: g.accessToken,
		},
	})
	if err != nil {
		log.Errorf("failed push commit to origin: %+v", err)
		return err
	}
	log.Infof("pushed commit %s", commitHash.String())
	return nil
}
