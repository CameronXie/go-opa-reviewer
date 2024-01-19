version:=1.0.0
stack_dir:=stack
go_code_dir:=internal pkg
dist_dir:=_dist
tests_dir:=${dist_dir}/tests

project_name:=go-opa-reviewer
github_app_secret_arn_param_name:=/$(project_name)/github-app-secret-arn
github_app_secret_id_param_name:=/$(project_name)/github-app-secret-id

# Docker
.PHONY: up
up: create-dev-env
	@docker compose up --build -d

.PHONY: down
down:
	@docker compose down -v

.PHONY: create-dev-env
create-dev-env:
	@test -e .env || cp .env.example .env

# CI/CD
.PHONY: deploy
deploy: test build deploy-secret deploy-app

.PHONY: deploy-secret
deploy-secret:
	@rain deploy stack/secret.yaml ${project_name}-secret -y \
		--params GitHubAppSecretArnParamName=$(github_app_secret_arn_param_name),GitHubAppSecretIdParamName=${github_app_secret_id_param_name}
	@aws secretsmanager put-secret-value \
		--secret-id ${project_name}-secret \
		--secret-string "{\"v3ApiUrl\":\"${GITHUB_V3_API_URL}\",\"integrationId\":${GITHUB_APP_INTEGRATION_ID},\"webhookSecret\":\"${GITHUB_APP_WEBHOOK_SECRET}\",\"privateKey\":\"${GITHUB_APP_PRIVATE_KEY}\"}"

.PHONY: deploy-app
deploy-app:
	@rain deploy stack/github-app.yaml ${project_name} -y \
		--params GitHubAppSecretArn=$(github_app_secret_arn_param_name),GitHubAppSecretId=${github_app_secret_id_param_name}

.PHONY: ci-test
ci-test: create-dev-env
	@docker compose run --rm dev sh -c 'make test'

# Dev
.PHONY: test
test: test-cfn test-policy test-lambda

.PHONY: build
build: cleanup-build build-policy build-lambda

.PHONY: cleanup-build
cleanup-build:
	@rm -rf ${dist_dir}
	@mkdir -p ${dist_dir}

## CFN
.PHONY: test-cfn
test-cfn: format-cfn lint-cfn

.PHONY: format-cfn
format-cfn:
	@rain fmt $(stack_dir)/*.yaml -w

.PHONY: lint-cfn
lint-cfn:
	@cfn-lint $(stack_dir)/*.yaml

## Policy
.PHONY: test-policy
test-policy:
	@opa test ./policy/ -v

.PHONY: build-policy
build-policy:
	@cd policy; opa build . -o ../${dist_dir}/bundle.tar.gz

## Lambda
.PHONY: test-lambda
test-lambda: lint-lambda lambda-unit

.PHONY: build-lambda
build-lambda:
	@go build -o ${dist_dir}/app \
		-a -ldflags '-X github.com/CameronXie/github-app-go-starter/internal/version.Version=${version} -extldflags "-s -w -static"' \
		cmd/app.go

.PHONY: lint-lambda
lint-lambda:
	@golangci-lint run $(addsuffix /..., $(go_code_dir)) -v

.PHONY: lambda-unit
lambda-unit:
	@rm -rf ${tests_dir}
	@mkdir -p ${tests_dir}
	@go clean -testcache
	@go test \
		-cover \
		-coverprofile=cp.out \
		-outputdir=${tests_dir} \
		-race \
		-v \
		-failfast \
		$(addprefix `pwd`/, $(addsuffix /..., $(go_code_dir)))
	@go tool cover -html=${tests_dir}/cp.out -o ${tests_dir}/cp.html
