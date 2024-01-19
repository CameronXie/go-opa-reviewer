package app

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/palantir/go-githubapp/githubapp"
	"github.com/stretchr/testify/assert"
)

type mockSecretsManagerClient struct {
	secretStr string
	err       error
}

func (m *mockSecretsManagerClient) GetSecretValue(
	_ context.Context,
	_ *secretsmanager.GetSecretValueInput,
	_ ...func(*secretsmanager.Options),
) (*secretsmanager.GetSecretValueOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	return &secretsmanager.GetSecretValueOutput{
		SecretString: aws.String(m.secretStr),
	}, nil
}

func TestGetAppConfigFromSecret(t *testing.T) {
	cases := map[string]struct {
		secretStr string
		secretErr error
		expected  *githubapp.Config
		errMsg    *string
	}{
		"get app config from secret": {
			secretStr: `
{
    "v3ApiUrl": "https://api.github.com/",
    "integrationId": 123456,
    "webhookSecret": "secret",
    "privateKey": "cHJpdmF0ZV9rZXk="
}`,
			expected: getGithubAppConfig(
				"https://api.github.com/",
				"secret",
				"private_key",
				123456,
			),
		},
		"fail to get secret should return error": {
			secretErr: errors.New("access denied"),
			errMsg:    aws.String("failed to fetch secret: access denied"),
		},
		"fail to unmarshal secret should return error": {
			secretStr: "{",
			errMsg:    aws.String("failed to unmarshal secret: unexpected end of JSON input"),
		},
		"fail to decode private key should return error": {
			secretStr: `
{
    "v3ApiUrl": "https://api.github.com/",
    "integrationId": 123456,
    "webhookSecret": "secret",
    "privateKey": "!@"
}
`,
			errMsg: aws.String("failed to decode private key: illegal base64 data at input byte 0"),
		},
	}

	for n, tc := range cases {
		t.Run(n, func(t *testing.T) {
			a := assert.New(t)
			appConfig, err := GetAppConfigFromSecret(
				context.Background(),
				&mockSecretsManagerClient{secretStr: tc.secretStr, err: tc.secretErr},
				"secretId",
			)

			if tc.errMsg != nil {
				a.Contains(err.Error(), *tc.errMsg)
				return
			}

			a.NoError(err)
			a.EqualValues(tc.expected, appConfig)
		})
	}
}

func TestGetPatternsFromCSV(t *testing.T) {
	cases := map[string]struct {
		csv      string
		expected []string
	}{
		"get patterns from csv": {
			csv:      "stacks/**/*.yaml,cfn/*.yaml",
			expected: []string{"stacks/**/*.yaml", "cfn/*.yaml"},
		},
		"get empty patterns from empty string": {
			csv:      "",
			expected: []string{},
		},
	}

	for n, tc := range cases {
		t.Run(n, func(t *testing.T) {
			a := assert.New(t)
			a.ElementsMatch(tc.expected, GetPatternsFromCSV(tc.csv))
		})
	}
}

func getGithubAppConfig(v3ApiURL, webhookSecret, privateKey string, integrationID int64) *githubapp.Config {
	cfg := new(githubapp.Config)
	cfg.V3APIURL = v3ApiURL
	cfg.App.IntegrationID = integrationID
	cfg.App.WebhookSecret = webhookSecret
	cfg.App.PrivateKey = privateKey

	return cfg
}
