package reviewer

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/open-policy-agent/opa/rego"
	"github.com/open-policy-agent/opa/util"
	"golang.org/x/net/context"
)

type Reviewer interface {
	Review(ctx context.Context, content []byte) ([]byte, error)
}

type reviewer struct {
	query rego.PreparedEvalQuery
}

// Review evaluates a given content using a prepared query and returns the results in JSON format.
func (r *reviewer) Review(ctx context.Context, content []byte) ([]byte, error) {
	var input any
	if err := util.Unmarshal(content, &input); err != nil {
		return nil, err
	}

	if input == nil {
		return nil, errors.New("failed to parse input")
	}

	results, queryErr := r.query.Eval(ctx, rego.EvalInput(input))
	if queryErr != nil {
		return nil, fmt.Errorf("failed to evaluate content: %w", queryErr)
	}

	resultJSON, _ := json.Marshal(results)
	return resultJSON, nil
}

// NewReviewerWithBundle initializes a new Reviewer implementation with a prepared query and returns it.
// It takes three parameters:
// - ctx: the context.Context to use for the evaluation process.
// - queryStr: the OPA query string to prepare for evaluation.
// - bundlePath: the path to the OPA bundle to load for evaluation.
// It returns a Reviewer interface and an error.
func NewReviewerWithBundle(ctx context.Context, queryStr, bundlePath string) (Reviewer, error) {
	query, err := rego.New(
		rego.Query(queryStr),
		rego.LoadBundle(bundlePath),
	).PrepareForEval(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to prepare the opa policy for evaluation: %w", err)
	}

	return &reviewer{query: query}, nil
}
