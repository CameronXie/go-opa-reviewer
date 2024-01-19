package review

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

func mockReadFileFun(_ context.Context, file string) ([]byte, error) {
	if strings.Contains(file, "invalid_read") {
		return nil, errors.New("access denied")
	}

	return []byte(file), nil
}

type mockReviewer struct {
}

func (m *mockReviewer) Review(_ context.Context, content []byte) ([]byte, error) {
	if strings.Contains(string(content), "invalid_review") {
		return nil, errors.New("invalid")
	}

	return []byte("valid"), nil
}

func TestService_Review(t *testing.T) {
	cases := map[string]struct {
		files           []string
		expectedResults []Result
		expectedErrMsg  *string
	}{
		"review files": {
			files: []string{
				"file_1",
				"invalid_read_file_2",
				"invalid_review_file_3",
			},
			expectedResults: []Result{
				{
					File:   "file_1",
					Output: []byte("valid"),
				},
				{
					File:  "invalid_read_file_2",
					Error: fmt.Errorf("failed to read file: %w", errors.New("access denied")),
				},
				{
					File:  "invalid_review_file_3",
					Error: fmt.Errorf("failed to review file: %w", errors.New("invalid")),
				},
			},
		},
	}

	for n, tc := range cases {
		t.Run(n, func(t *testing.T) {
			a := assert.New(t)
			svc, err := New(new(mockReviewer), 1, 1)
			a.NoError(err)

			results, err := svc.Review(context.TODO(), mockReadFileFun, tc.files)
			if tc.expectedErrMsg != nil {
				a.Contains(err.Error(), *tc.expectedErrMsg)
				return
			}

			a.NoError(err)
			a.ElementsMatch(tc.expectedResults, results)
		})
	}
}
