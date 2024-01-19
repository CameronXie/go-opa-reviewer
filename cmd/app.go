package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/CameronXie/go-opa-reviewer/internal/app"
	"github.com/CameronXie/go-opa-reviewer/internal/prhandler"
	"github.com/CameronXie/go-opa-reviewer/internal/review"
	"github.com/CameronXie/go-opa-reviewer/internal/version"
	"github.com/CameronXie/go-opa-reviewer/pkg/reviewer"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/awslabs/aws-lambda-go-api-proxy/httpadapter"
	"github.com/palantir/go-githubapp/githubapp"
	"github.com/rs/zerolog"
)

const (
	appName          = "go-opa-reviewer"
	bundlePath       = "/opt/bundle.tar.gz"
	readerPoolSize   = 100
	reviewerPoolSize = 100
	secretIdEnv      = "GITHUB_APP_SECRET_ID"
	policyQueryEnv   = "GITHUB_APP_POLICY_QUERY"
	filePatterns     = "GITHUB_APP_FILE_PATTERNS"
	logLevel         = zerolog.DebugLevel
)

func main() {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	zerolog.DefaultContextLogger = &logger

	awsConfig, awsConfigErr := config.LoadDefaultConfig(context.TODO())
	checkError(awsConfigErr)

	appConfig, configErr := app.GetAppConfigFromSecret(
		context.Background(),
		secretsmanager.NewFromConfig(awsConfig),
		os.Getenv(secretIdEnv),
	)
	checkError(configErr)

	githubClientCreator, clientErr := githubapp.NewDefaultCachingClientCreator(
		*appConfig,
		githubapp.WithClientUserAgent(fmt.Sprintf("%s:%s", appName, version.Version)),
		githubapp.WithClientTimeout(3*time.Second),
		githubapp.WithClientMiddleware(
			githubapp.ClientLogging(logLevel),
		),
	)
	checkError(clientErr)

	fileReviewer, reviewerErr := reviewer.NewReviewerWithBundle(
		context.Background(),
		os.Getenv(policyQueryEnv),
		bundlePath,
	)
	checkError(reviewerErr)

	reviewSvc, svcErr := review.New(fileReviewer, readerPoolSize, reviewerPoolSize)
	checkError(svcErr)

	webhookHandler := githubapp.NewDefaultEventDispatcher(
		*appConfig,
		prhandler.New(githubClientCreator, app.GetPatternsFromCSV(os.Getenv(filePatterns)), reviewSvc),
	)

	http.Handle(githubapp.DefaultWebhookRoute, webhookHandler)
	lambda.Start(httpadapter.NewALB(http.DefaultServeMux).ProxyWithContext)
}

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
