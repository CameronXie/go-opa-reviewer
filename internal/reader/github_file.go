package reader

import (
	"context"

	"github.com/google/go-github/v58/github"
)

// ReadGitHubFile fetches the content of a file from a GitHub repository.
func ReadGitHubFile(client *github.Client, owner, repo, ref string) func(context.Context, string) ([]byte, error) {
	return func(ctx context.Context, fileName string) ([]byte, error) {
		file, _, _, err := client.Repositories.GetContents(
			ctx, owner, repo, fileName, &github.RepositoryContentGetOptions{Ref: ref},
		)
		if err != nil {
			return nil, err
		}

		content, err := file.GetContent()
		if err != nil {
			return nil, err
		}

		return []byte(content), nil
	}
}
