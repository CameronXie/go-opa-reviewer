package reviewer

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReview(t *testing.T) {
	cases := map[string]struct {
		input     string
		expected  string
		query     string
		reviewErr error
		willPanic bool
	}{
		"review valid yaml input": {
			input: `
Resources:
  SecurityGroupA:
    Type: AWS::EC2::SecurityGroup
    Properties:
      GroupDescription: Allow HTTP
      VpcId: !Ref VPC
      SecurityGroupIngress:
        - IpProtocol: tcp
          FromPort: 80
          ToPort: 80
          CidrIp: 0.0.0.0/0
  SecurityGroupB:
    Type: AWS::EC2::SecurityGroup
    Properties:
      GroupDescription: Allow HTTPS
      VpcId: !Ref VPC
      SecurityGroupIngress:
        - IpProtocol: tcp
          FromPort: 443
          ToPort: 443
          CidrIp: 10.0.0.0/25
`,
			query:    "data.reviewer.cfn",
			expected: `[{"expressions":[{"value":{"allow":false,"violation":["SecurityGroupA"]},"text":"data.reviewer.cfn","location":{"row":1,"col":1}}]}]`, // nolint: lll
		},
		"review empty input": {
			input:     "",
			query:     "data",
			reviewErr: errors.New("failed to parse input"),
		},
		"review invalid yaml syntax input": {
			input:     `'invalid_yaml`,
			query:     "data",
			reviewErr: errors.New("yaml: found unexpected end of stream"),
		},
		"review invalid query": {
			input:     `{}`,
			query:     "random",
			willPanic: true,
		},
	}

	for n, tc := range cases {
		t.Run(n, func(t *testing.T) {
			a := assert.New(t)
			r, _ := NewReviewerWithBundle(context.TODO(), tc.query, "testdata/bundle.tar.gz")

			if tc.willPanic {
				a.Panics(func() {
					_, _ = r.Review(context.TODO(), []byte(tc.input))
				})
				return
			}

			result, err := r.Review(context.TODO(), []byte(tc.input))
			a.Equal(tc.reviewErr, err)
			a.Equal(tc.expected, string(result))
		})
	}
}

func TestNewReviewerWithBundle(t *testing.T) {
	cases := map[string]struct {
		bundle    string
		willPanic bool
	}{
		"with valid bundle path": {
			bundle: "testdata/bundle.tar.gz",
		},
		"with invalid bundle path": {
			bundle: "invalid_bundle_path",
		},
	}

	for n, tc := range cases {
		t.Run(n, func(t *testing.T) {
			a := assert.New(t)
			query := "data"

			if tc.willPanic {
				a.Panics(func() {
					_, _ = NewReviewerWithBundle(context.TODO(), query, tc.bundle)
				})
				return
			}

			a.NotPanics(func() {
				_, _ = NewReviewerWithBundle(context.TODO(), query, tc.bundle)
			})
		})
	}
}
