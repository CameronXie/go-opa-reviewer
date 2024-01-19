package app

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/palantir/go-githubapp/githubapp"
)

type SecretsManagerClient interface {
	GetSecretValue(
		ctx context.Context,
		params *secretsmanager.GetSecretValueInput,
		optFns ...func(*secretsmanager.Options),
	) (*secretsmanager.GetSecretValueOutput, error)
}

type Config struct {
	V3ApiURL      string `json:"v3ApiUrl"`
	IntegrationID int64  `json:"integrationId"`
	WebhookSecret string `json:"webhookSecret"`
	PrivateKey    string `json:"privateKey"`
}

// GetAppConfigFromSecret retrieves the application configuration from a secret using the provided SecretsManagerClient.
// It returns the config as a githubapp.Config pointer and an error
func GetAppConfigFromSecret(
	ctx context.Context,
	client SecretsManagerClient,
	secretID string,
) (*githubapp.Config, error) {
	secret, secretErr := client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{SecretId: aws.String(secretID)})
	if secretErr != nil {
		return nil, fmt.Errorf("failed to fetch secret: %w", secretErr)
	}

	cfg := new(Config)
	jsonErr := json.Unmarshal([]byte(*secret.SecretString), cfg)
	if jsonErr != nil {
		return nil, fmt.Errorf("failed to unmarshal secret: %w", jsonErr)
	}

	privateKey, decodeErr := base64.StdEncoding.DecodeString(cfg.PrivateKey)
	if decodeErr != nil {
		return nil, fmt.Errorf("failed to decode private key: %w", decodeErr)
	}

	appConfig := new(githubapp.Config)
	appConfig.V3APIURL = cfg.V3ApiURL
	appConfig.App.IntegrationID = cfg.IntegrationID
	appConfig.App.WebhookSecret = cfg.WebhookSecret
	appConfig.App.PrivateKey = string(privateKey)

	return appConfig, nil
}

// GetPatternsFromCSV retrieves patterns from a CSV string and returns them as a slice of strings.
func GetPatternsFromCSV(csv string) []string {
	patterns := make([]string, 0)
	for _, pattern := range strings.Split(csv, ",") {
		if pattern == "" {
			continue
		}

		patterns = append(patterns, pattern)
	}

	return patterns
}
