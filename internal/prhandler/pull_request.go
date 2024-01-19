package prhandler

import (
	"fmt"

	"github.com/google/go-github/v58/github"
)

type pullRequest struct {
	num            int
	repo           *github.Repository
	sha            string
	action         string
	installationID int64
}

func (pr pullRequest) getPullRequestString() string {
	return fmt.Sprintf("%s#%d", pr.repo.GetFullName(), pr.num)
}

func (pr pullRequest) getOwner() string {
	return pr.repo.GetOwner().GetLogin()
}

func (pr pullRequest) getRepoName() string {
	return pr.repo.GetName()
}
