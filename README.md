# Go OPA Reviewer

[![Test](https://github.com/CameronXie/go-opa-reviewer/actions/workflows/test.yaml/badge.svg?branch=main)](https://github.com/CameronXie/go-opa-reviewer/actions/workflows/test.yaml)

This project demonstrates an approach for creating a GitHub App that leverages the Open Policy Agent (OPA) to perform
queries on the modified files present in a Pull Request.

This project is built with Golang (v1.21) and hosted with AWS Lambda which fronted by AWS ALB. The project is using
Lambda Layer to store the policy bundler, and use Secrets Manager to store GitHub App configurations and secrets.

## Design

![Go OPA Reviewer](./images/reviewer_design.png "Go OPA Reviewer")

1. A Pull Request event is initiated on GitHub and delivery to the Application Load Balancer.
2. The Application Load Balancer invoke the Reviewer Lambda function.
3. The Reviewer Lambda function then retrieves the content of the modified files from GitHub and perform policy checks.
4. The review results are posted back to the Pull Request page as comment.

## Prerequisites

* Register
  a GitHub App with following permissions and subscribing events, and
  update `GITHUB_V3_API_URL`, `GITHUB_APP_INTEGRATION_ID`, `GITHUB_APP_WEBHOOK_SECRET` and `GITHUB_APP_PRIVATE_KEY`
  in  `.env` file. `GITHUB_APP_PRIVATE_KEY` value should be base64 encoded.
    * Repository Permissions:
        * Content: Read-only
        * Pull requests: Read and write
        * Metadata: Read-only
    * Subscribe to events:
        * Pull request
* An AWS account which has sufficient permission to deploy VPC, ALB, Lambda and SecretsManager.
* Docker and Docker Compose installed.

## Folder Structure

```shell
.
├── Makefile
├── README.md
├── cmd
│   └── app.go               # Entry point to the Reviewer GitHub App.
├── docker
│   └── dev
├── docker-compose.yml
├── go.mod
├── go.sum
├── image
├── internal
│   ├── app                  # The main GitHub App package.
│   ├── presentation         # Handles the presentation of the review results. 
│   ├── prhandler            # Manages the handling of pull request events.
│   ├── reader               # Provides functionality for reading files.
│   ├── review               # Review service.
│   └── version              # Manages the project version.
├── pkg
│   └── reviewer             # Integrated with OPA SDK and handles the policy review.
├── policy                   # Contains Rego and Rego test files.
│   ├── main.rego
│   └── main_test.rego
├── stack                    # CloudFormation templates.
│   ├── github-app.yaml
│   └── secret.yaml
└── vendor
```

## Deploy

* This project utilizes Docker to manage the local development environment. Execute the `make up` command to start the
  development container.
* Update the `.env` file as per your requirements.
* From within the `go_opa_reviewer_dev` container, execute `make deploy` which will first initiate testings and linting
  for CloudFormation templates, Policy (Rego) files and the Lambda code, build policy bundle and lambda executable, and
  deploy to your aws account.
* Once the deployment process is finished, retrieve the webhook URL from the CloudFormation output and update it on your
  GitHub App configuration page.

## Test

Run the `make test` command in the `go_opa_reviewer_dev` container. This command will initiate testing and linting
processes for CloudFormation files, Policy (Rego) files, and the Lambda code.
