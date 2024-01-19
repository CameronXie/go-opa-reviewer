package presentation

import (
	"errors"
	"testing"

	"github.com/CameronXie/go-opa-reviewer/internal/review"
	"github.com/stretchr/testify/assert"
)

func TestMarkdown(t *testing.T) {
	cases := map[string]struct {
		results  []review.Result
		expected string
	}{
		"no results": {
			results:  make([]review.Result, 0),
			expected: "no review results available",
		},
		"display results and errors": {
			results: []review.Result{
				{
					File:   "file-1",
					Output: []byte("outcome_1"),
				},
				{
					File:   "file-2",
					Output: []byte("outcome_2"),
				},
				{
					File:  "file-3",
					Error: errors.New("error_1"),
				},
			},
			expected: `Reviews:
* file-1: outcome_1
* file-2: outcome_2

Errors:
* file-3: error_1
`,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			a := assert.New(t)
			actual := Markdown(tc.results)
			a.Equal(tc.expected, actual)
		})
	}
}
