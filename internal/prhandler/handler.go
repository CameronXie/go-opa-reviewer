package prhandler

import (
	"context"
	"fmt"

	"github.com/CameronXie/go-opa-reviewer/internal/presentation"
	"github.com/CameronXie/go-opa-reviewer/internal/reader"
	"github.com/CameronXie/go-opa-reviewer/internal/review"
	"github.com/bmatcuk/doublestar"
	"github.com/google/go-github/v58/github"
	"github.com/palantir/go-githubapp/githubapp"
)

const (
	pullRequestEvent  = "pull_request"
	numResultsPerPage = 30
)

type handler struct {
	eventActivityTypes []string
	patterns           []string
	clientCreator      githubapp.ClientCreator
	reviewSvc          review.Service
}

func (h *handler) Handles() []string {
	return []string{pullRequestEvent}
}

func (h *handler) Handle(ctx context.Context, eventType, _ string, payload []byte) error {
	pr, eventErr := parsePullRequestEvent(eventType, payload)
	if eventErr != nil {
		return eventErr
	}

	ctx, logger := githubapp.PreparePRContext(ctx, pr.installationID, pr.repo, pr.num)

	if action := pr.action; !contains(h.eventActivityTypes, action) {
		logger.Info().Msgf("received action %s, no further processing is required", action)
		return nil
	}

	client, clientErr := h.clientCreator.NewInstallationClient(pr.installationID)
	if clientErr != nil {
		return clientErr
	}

	logger.Debug().Msgf("fetching changed files from %s", pr.getPullRequestString())
	files, filesErr := getChangedFiles(ctx, client, pr.getOwner(), pr.getRepoName(), pr.num)
	if filesErr != nil {
		return filesErr
	}

	fileNames := getMatchingFileNames(files, h.patterns)
	if len(fileNames) == 0 {
		return postComment(ctx, client, pr.getOwner(), pr.getRepoName(), pr.num, "no files matched the provided patterns")
	}

	logger.Debug().Msgf("reviewing %d changed files from %s", len(fileNames), pr.getPullRequestString())
	results, reviewErr := h.reviewSvc.Review(
		ctx,
		reader.ReadGitHubFile(client, pr.getOwner(), pr.getRepoName(), pr.sha),
		fileNames,
	)
	if reviewErr != nil {
		return reviewErr
	}

	logger.Debug().Msgf("posting comment on %s", pr.getPullRequestString())
	return postComment(ctx, client, pr.getOwner(), pr.getRepoName(), pr.num, presentation.Markdown(results))
}

// parsePullRequestEvent parses a pull request event and returns the corresponding *pullRequest object
// along with an error if any.
func parsePullRequestEvent(eventType string, payload []byte) (*pullRequest, error) {
	event, err := github.ParseWebHook(eventType, payload)
	if err != nil {
		return nil, err
	}

	prEvent, ok := event.(*github.PullRequestEvent)
	if !ok {
		return nil, fmt.Errorf("unexpected event payload type %s found", eventType)
	}

	return &pullRequest{
		num:            prEvent.GetPullRequest().GetNumber(),
		repo:           prEvent.GetRepo(),
		sha:            prEvent.GetPullRequest().GetHead().GetSHA(),
		action:         prEvent.GetAction(),
		installationID: prEvent.GetInstallation().GetID(),
	}, nil
}

// getChangedFiles retrieves the list of changed files in a pull request.
func getChangedFiles(
	ctx context.Context,
	client *github.Client,
	owner string,
	repo string,
	num int,
) ([]*github.CommitFile, error) {
	opt := &github.ListOptions{
		PerPage: numResultsPerPage,
	}

	commitFiles := make([]*github.CommitFile, 0)
	for {
		files, resp, err := client.PullRequests.ListFiles(ctx, owner, repo, num, opt)
		if err != nil {
			return nil, err
		}

		commitFiles = append(commitFiles, files...)

		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	return commitFiles, nil
}

// getMatchingFileNames returns a list of file names from the given list of commit files
// that match any of the provided patterns.
func getMatchingFileNames(files []*github.CommitFile, patterns []string) []string {
	fileNames := make([]string, 0)
	for idx := range files {
		fileName := files[idx].GetFilename()
		if isMatchedFile(fileName, patterns) {
			fileNames = append(fileNames, fileName)
		}
	}

	return fileNames
}

// isMatchedFile checks if a given file name matches any of the glob patterns provided.
func isMatchedFile(fileName string, patterns []string) bool {
	for _, path := range patterns {
		if matched, _ := doublestar.PathMatch(path, fileName); matched {
			return true
		}
	}

	return false
}

func postComment(
	ctx context.Context,
	client *github.Client,
	owner string,
	repo string,
	num int,
	content string,
) error {
	_, _, err := client.Issues.CreateComment(ctx, owner, repo, num, &github.IssueComment{
		Body: github.String(content),
	})

	return err
}
func contains[T comparable](s []T, item T) bool {
	for idx := range s {
		if s[idx] == item {
			return true
		}
	}

	return false
}

func New(clientCreator githubapp.ClientCreator, patterns []string, reviewSvc review.Service) githubapp.EventHandler {
	return &handler{
		clientCreator:      clientCreator,
		patterns:           patterns,
		reviewSvc:          reviewSvc,
		eventActivityTypes: []string{"opened", "reopened", "synchronize", "ready_for_review"},
	}
}
