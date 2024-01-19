package reader

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/go-github/v58/github"
	"github.com/migueleliasweb/go-github-mock/src/mock"
	"github.com/stretchr/testify/assert"
)

func TestReadGitHubFile(t *testing.T) {
	cases := map[string]struct {
		content  string
		encoding *string
		fileErr  bool
		expected string
		errMsg   *string
	}{
		"get file content": {
			content:  "file_content",
			expected: "file_content",
		},
		"get base64 encoded file content": {
			content:  "ZmlsZV9jb250ZW50",
			encoding: github.String("base64"),
			expected: "file_content",
		},
		"failed to get file should return error": {
			content: "file_content",
			fileErr: true,
			errMsg:  github.String("/repos/owner/repo/contents/file?ref=ref: 400 bad request"),
		},
		"failed to get file content should return error": {
			content:  "file_content",
			encoding: github.String("random"),
			errMsg:   github.String("unsupported content encoding: random"),
		},
	}

	for n, tc := range cases {
		t.Run(n, func(t *testing.T) {
			a := assert.New(t)
			mockContentResp := mock.WithRequestMatch(
				mock.GetReposContentsByOwnerByRepoByPath,
				&github.RepositoryContent{
					Content:  github.String(tc.content),
					Encoding: tc.encoding,
				},
			)

			if tc.fileErr {
				mockContentResp = mock.WithRequestMatchHandler(
					mock.GetReposContentsByOwnerByRepoByPath,
					http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
						mock.WriteError(
							w,
							http.StatusBadRequest,
							"bad request",
						)
					}),
				)
			}

			bs, err := ReadGitHubFile(
				github.NewClient(mock.NewMockedHTTPClient(mockContentResp)), "owner", "repo", "ref",
			)(context.TODO(), "file")

			if tc.errMsg != nil {
				a.Contains(err.Error(), *tc.errMsg)
				return
			}

			a.NoError(err)
			a.Equal(tc.expected, string(bs))
		})
	}
}
