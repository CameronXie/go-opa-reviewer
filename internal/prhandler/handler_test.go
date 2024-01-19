package prhandler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/CameronXie/go-opa-reviewer/internal/review"
	"github.com/google/go-github/v58/github"
	"github.com/migueleliasweb/go-github-mock/src/mock"
	"github.com/palantir/go-githubapp/githubapp"
	"github.com/stretchr/testify/assert"
)

type mockClientCreator struct {
	githubapp.ClientCreator
	client    *github.Client
	clientErr error
}

func (m *mockClientCreator) NewInstallationClient(_ int64) (*github.Client, error) {
	if m.clientErr != nil {
		return nil, m.clientErr
	}

	return m.client, nil
}

type mockReviewSvc struct {
	err error
}

func (m *mockReviewSvc) Review(_ context.Context, _ review.ReadFileFunc, files []string) ([]review.Result, error) {
	if m.err != nil {
		return nil, m.err
	}

	res := make([]review.Result, 0)
	for _, file := range files {
		if strings.Contains(file, "invalid") {
			res = append(res, review.Result{
				File:  file,
				Error: errors.New("invalid file"),
			})
			continue
		}

		res = append(res, review.Result{
			File:   file,
			Output: []byte("valid file"),
		})
	}

	return res, m.err
}

func TestHandler_Handle(t *testing.T) {
	cases := map[string]struct {
		payload             []byte
		installClientErr    error
		listChangedFiles    []string
		listChangedFilesErr bool
		reviewErr           error
		expectedComment     string
		expectedErrMsg      *string
	}{
		"review files and post results in comment": {
			payload: getPullRequestPayload("opened"),
			listChangedFiles: []string{
				"file_1.yaml",
				"stack/file_2.yaml",
				"stack/app/invalid_file_3.yaml",
			},
			expectedComment: "{\"body\":\"Reviews:\\n* stack/file_2.yaml: valid file\\n\\nErrors:\\n* stack/app/invalid_file_3.yaml: invalid file\\n\"}\n", // nolint: lll
		},
		"no matching file found and post msg in comment": {
			payload:          getPullRequestPayload("synchronize"),
			listChangedFiles: []string{"file_1.yaml"},
			expectedComment:  "{\"body\":\"no files matched the provided patterns\"}\n",
		},
		"ignore untrack event type and not post msg in comment": {
			payload:          getPullRequestPayload("labeled"),
			listChangedFiles: []string{"stack/file_1.yaml"},
			expectedComment:  "",
		},
		"failed to parse event payload should return error": {
			payload:        []byte(`{`),
			expectedErrMsg: strPtr("unexpected end of JSON input"),
		},
		"failed to init github client should return error": {
			payload:          getPullRequestPayload("opened"),
			installClientErr: errors.New("failed to setup client"),
			expectedErrMsg:   strPtr("failed to setup client"),
		},
		"failed to list commit files should return error": {
			payload:             getPullRequestPayload("opened"),
			listChangedFilesErr: true,
			expectedErrMsg:      strPtr("repos/owner/repo/pulls/2/files?per_page=30: 400 bad request"),
		},
		"failed to review files should return error": {
			payload:          getPullRequestPayload("opened"),
			listChangedFiles: []string{"stack/file_1.yaml"},
			reviewErr:        errors.New("failed to review files"),
			expectedErrMsg:   strPtr("failed to review files"),
		},
	}

	for n, tc := range cases {
		t.Run(n, func(t *testing.T) {
			a := assert.New(t)
			var sb strings.Builder

			listFileMock := mock.WithRequestMatch(
				mock.GetReposPullsFilesByOwnerByRepoByPullNumber,
				toCommitFiles(tc.listChangedFiles),
			)

			if tc.listChangedFilesErr {
				listFileMock = mock.WithRequestMatchHandler(
					mock.GetReposPullsFilesByOwnerByRepoByPullNumber,
					http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
						mock.WriteError(
							w,
							http.StatusBadRequest,
							"bad request",
						)
					}),
				)
			}

			client := github.NewClient(mock.NewMockedHTTPClient(
				listFileMock,
				mock.WithRequestMatchHandler(
					mock.PostReposIssuesCommentsByOwnerByRepoByIssueNumber,
					http.HandlerFunc(func(_ http.ResponseWriter, req *http.Request) {
						defer req.Body.Close()
						content, _ := io.ReadAll(req.Body)
						sb.Write(content)
					}),
				),
			))

			h := New(
				&mockClientCreator{client: client, clientErr: tc.installClientErr},
				[]string{"stack/**/*.yaml"},
				&mockReviewSvc{err: tc.reviewErr},
			)

			a.EqualValues([]string{pullRequestEvent}, h.Handles())
			err := h.Handle(context.TODO(), pullRequestEvent, "", tc.payload)

			if tc.expectedErrMsg != nil {
				a.Contains(err.Error(), *tc.expectedErrMsg)
				return
			}

			a.Nil(err)
			a.Equal(tc.expectedComment, sb.String())
		})
	}
}

func getPullRequestPayload(action string) []byte {
	payload := map[string]any{
		"action": action,
		"number": 1,
		"pull_request": map[string]any{
			"number": 2,
			"head": map[string]any{
				"sha": "12345",
				"user": map[string]any{
					"login": "owner",
				},
			},
		},
		"repository": map[string]any{
			"id":        12345678,
			"name":      "repo",
			"full_name": "owner/repo",
			"owner": map[string]any{
				"login": "owner",
			},
		},
		"installation": map[string]any{
			"id": 12345678,
		},
	}

	bs, _ := json.Marshal(payload)
	return bs
}

func toCommitFiles(s []string) []*github.CommitFile {
	commitFiles := make([]*github.CommitFile, 0, len(s))
	for _, file := range s {
		commitFiles = append(commitFiles, &github.CommitFile{Filename: github.String(file)})
	}
	return commitFiles
}

func strPtr(str string) *string {
	return &str
}
