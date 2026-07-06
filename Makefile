PACKAGE=github.com/useryege/athena/common
CURRENT_DIR=$(shell pwd)
DIST_DIR=${CURRENT_DIR}/dist
BIN_NAME=athena

UNAME_S:=$(shell uname)
IS_DARWIN:=$(if $(filter Darwin, $(UNAME_S)),true,false)

# When using OSX/Darwin, you might need to enable CGO for local builds
DEFAULT_CGO_FLAG:=0
ifeq ($(IS_DARWIN),true)
    DEFAULT_CGO_FLAG:=1
endif
CGO_FLAG?=${DEFAULT_CGO_FLAG}

GOOS?=$(shell go env GOOS)
GOARCH?=$(shell go env GOARCH)

TARGET_ARCH?=linux/amd64
E2E_GO_TEST_FLAGS ?= -count=1
E2E_PACKAGES ?= $(shell go list ./e2e/tests/... 2>/dev/null | grep -v '/live/')
E2E_LIVE_PACKAGES ?= ./e2e/tests/live/...

VERSION=$(shell cat ${CURRENT_DIR}/VERSION)
BUILD_DATE:=$(if $(BUILD_DATE),$(BUILD_DATE),$(shell date -u +'%Y-%m-%dT%H:%M:%SZ'))
GIT_COMMIT:=$(if $(GIT_COMMIT),$(GIT_COMMIT),$(shell git rev-parse HEAD))
GIT_TAG:=$(if $(GIT_TAG),$(GIT_TAG),$(shell if [ -z "`git status --porcelain`" ]; then git describe --exact-match --tags HEAD 2>/dev/null; fi))
GIT_TREE_STATE:=$(if $(GIT_TREE_STATE),$(GIT_TREE_STATE),$(shell if [ -z "`git status --porcelain`" ]; then echo "clean" ; else echo "dirty"; fi))

# Docker command to use
DOCKER ?= docker
ifneq ($(DOCKER),docker)
$(error Only Docker is supported. Please run make with DOCKER=docker)
endif

# pointing to python 3.12 
MKDOCS_DOCKER_IMAGE?=python:3.12-alpine
MKDOCS_RUN_ARGS?=

PATH:=$(PATH):$(PWD)/hack

PROD_IMAGE?=athena:local
PROD_COMPOSE_FILE?=docker-compose.prod.yml
PROD_ENV_FILE?=.env.prod
REMOTE_APP_DIR?=/root/athena
REMOTE_USER?=root
PROD_LOG_SERVICE?=
PROD_MIGRATE_MODULE?=all
PROD_POSTGRES_VOLUME?=athena-prod-postgres-data
PROD_COMPOSE_LOCAL=PROD_IMAGE=$(PROD_IMAGE) PROD_POSTGRES_VOLUME=$(PROD_POSTGRES_VOLUME) $(DOCKER) compose -f $(PROD_COMPOSE_FILE) --env-file $(PROD_ENV_FILE)
# perform static compilation
DEFAULT_STATIC_BUILD:=true
ifeq ($(IS_DARWIN),true)
    DEFAULT_STATIC_BUILD:=false
endif
STATIC_BUILD?=${DEFAULT_STATIC_BUILD}

override LDFLAGS += \
  -X ${PACKAGE}.version=${VERSION} \
  -X ${PACKAGE}.buildDate=${BUILD_DATE} \
  -X ${PACKAGE}.gitCommit=${GIT_COMMIT} \
  -X ${PACKAGE}.gitTreeState=${GIT_TREE_STATE}\
  -X "${PACKAGE}.extraBuildInfo=${EXTRA_BUILD_INFO}"

ifeq (${STATIC_BUILD}, true)
override LDFLAGS += -extldflags "-static"
endif


ifneq (${GIT_TAG},)
override LDFLAGS += -X ${PACKAGE}.gitTag=${GIT_TAG}
endif

# Installs all tools required for running codegen (Linux packages)
.PHONY: install-codegen-tools-local
install-codegen-tools-local:
	./hack/install.sh codegen-tools
	./hack/install.sh codegen-go-tools

.PHONY: mockgen
mockgen:
	./hack/generate-mock.sh

.PHONY: gogen
gogen:
	export GO111MODULE=off
	go generate ./...

.PHONY: protogen
protogen: mod-vendor-local protogen-fast

.PHONY: protogen-fast
protogen-fast:
	export GO111MODULE=off
	./hack/generate-proto.sh

.PHONY: sqlc-local
sqlc-local:
	go run -mod=mod github.com/sqlc-dev/sqlc/cmd/sqlc generate

.PHONY: abigen-local
abigen-local:
	./hack/generate-abi.sh

.PHONY: clientgen
clientgen:
	export GO111MODULE=off
	./hack/update-codegen.sh

.PHONY: clidocsgen
clidocsgen:
	go run tools/cmd-docs/main.go

.PHONY: password-hash
password-hash:
ifeq ($(PASSWORD),)
	go run tools/password-hash/main.go
else
	go run tools/password-hash/main.go -password '$(PASSWORD)'
endif

.PHONY: jwt-secret
jwt-secret:
	@go run tools/jwt-secret/main.go

.PHONY: service-password
service-password:
	@go run tools/service-password/main.go

.PHONY: wallet-private-key-ciphertext
wallet-private-key-ciphertext:
	@go run tools/wallet-private-key-ciphertext/main.go $(ARGS)

.PHONY: prod-reset-secrets
prod-reset-secrets:
	@go run tools/prod-env-reset/main.go -env-file $(PROD_ENV_FILE)

.PHONY: auto-clicker
auto-clicker:
	@mkdir -p ${DIST_DIR}
	@GOOS=windows GOARCH=amd64 go build -o ${DIST_DIR}/auto-clicker.exe ./tools/auto-clicker

.PHONY: mod-download-local
mod-download-local:
	go mod download && go mod tidy

.PHONY: mod-vendor-local
mod-vendor-local: mod-download-local
	go mod vendor

# new codegen-local
.PHONY: codegen-local
codegen-local: mod-vendor-local mockgen gogen protogen sqlc-local clientgen clidocsgen
	rm -rf vendor/

# Cleans VSCode debug.test files from sub-dirs to prevent them from being included in by golang embed
.PHONY: clean-debug
clean-debug:
	-find ${CURRENT_DIR} -name debug.test -exec rm -f {} +

.PHONY: athena-all
athena-all: clean-debug
	CGO_ENABLED=${CGO_FLAG} GOOS=${GOOS} GOARCH=${GOARCH} GODEBUG="tarinsecurepath=0,zipinsecurepath=0" go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/${BIN_NAME} ./cmd

# Run goreman start with exclude option , provide exclude env variable with list of services
.PHONY: run
run:
	bash ./hack/goreman-start.sh

.PHONY: e2e
e2e:
	go test $(E2E_GO_TEST_FLAGS) $(E2E_PACKAGES)

.PHONY: e2e-ethereumapi
e2e-ethereumapi:
	go test $(E2E_GO_TEST_FLAGS) ./e2e/tests/ethereumapi

.PHONY: e2e-live
e2e-live:
	@if [ "$${E2E_LIVE:-}" != "1" ]; then \
		printf '%s\n' 'Refusing to run live E2E tests without explicit confirmation.' >&2; \
		printf '%s\n' 'Run: E2E_LIVE=1 make e2e-live' >&2; \
		exit 1; \
	fi
	go test $(E2E_GO_TEST_FLAGS) $(E2E_LIVE_PACKAGES)

.PHONY: e2e-live-ethereumapi
e2e-live-ethereumapi:
	@if [ "$${E2E_LIVE:-}" != "1" ]; then \
		printf '%s\n' 'Refusing to run ethereum-api live E2E tests without explicit confirmation.' >&2; \
		printf '%s\n' 'Run: E2E_LIVE=1 make e2e-live-ethereumapi' >&2; \
		exit 1; \
	fi
	go test $(E2E_GO_TEST_FLAGS) ./e2e/tests/live/ethereumapi

.PHONY: e2e-live-etherscan-rate-limit
e2e-live-etherscan-rate-limit:
	@if [ "$${E2E_LIVE:-}" != "1" ]; then \
		printf '%s\n' 'Refusing to run Etherscan rate-limit probe without live test confirmation.' >&2; \
		printf '%s\n' 'Run: E2E_LIVE=1 ATHENA_E2E_ETHERSCAN_RATE_LIMIT_PROBE=1 make e2e-live-etherscan-rate-limit' >&2; \
		exit 1; \
	fi
	@if [ "$${ATHENA_E2E_ETHERSCAN_RATE_LIMIT_PROBE:-}" != "1" ]; then \
		printf '%s\n' 'Refusing to intentionally trigger Etherscan rate limiting without probe confirmation.' >&2; \
		printf '%s\n' 'Run: E2E_LIVE=1 ATHENA_E2E_ETHERSCAN_RATE_LIMIT_PROBE=1 make e2e-live-etherscan-rate-limit' >&2; \
		exit 1; \
	fi
	go test $(E2E_GO_TEST_FLAGS) ./e2e/tests/live/ethereumapi -run TestEtherscanFreePlanRateLimitProbe -v

.PHONY: e2e-live-etherscan-multi-key-rate-limit
e2e-live-etherscan-multi-key-rate-limit:
	@if [ "$${E2E_LIVE:-}" != "1" ]; then \
		printf '%s\n' 'Refusing to run Etherscan multi-key probe without live test confirmation.' >&2; \
		printf '%s\n' 'Run: E2E_LIVE=1 ATHENA_E2E_ETHERSCAN_MULTI_KEY_PROBE=1 ATHENA_E2E_ETHERSCAN_API_KEYS=key1,key2,key3 make e2e-live-etherscan-multi-key-rate-limit' >&2; \
		exit 1; \
	fi
	@if [ "$${ATHENA_E2E_ETHERSCAN_MULTI_KEY_PROBE:-}" != "1" ]; then \
		printf '%s\n' 'Refusing to intentionally run the Etherscan multi-key probe without confirmation.' >&2; \
		printf '%s\n' 'Run: E2E_LIVE=1 ATHENA_E2E_ETHERSCAN_MULTI_KEY_PROBE=1 ATHENA_E2E_ETHERSCAN_API_KEYS=key1,key2,key3 make e2e-live-etherscan-multi-key-rate-limit' >&2; \
		exit 1; \
	fi
	go test $(E2E_GO_TEST_FLAGS) ./e2e/tests/live/ethereumapi -run TestEtherscanMultiKeyAggregateRateLimitProbe -v

.PHONY: e2e-live-etherscan-proxy-multi-key-rate-limit
e2e-live-etherscan-proxy-multi-key-rate-limit:
	@if [ "$${E2E_LIVE:-}" != "1" ]; then \
		printf '%s\n' 'Refusing to run Etherscan proxy multi-key probe without live test confirmation.' >&2; \
		printf '%s\n' 'Run: E2E_LIVE=1 ATHENA_E2E_ETHERSCAN_PROXY_MULTI_KEY_PROBE=1 ATHENA_E2E_ETHERSCAN_API_KEYS=key1,key2 ATHENA_E2E_ETHERSCAN_PROXY_URLS=http://proxy1:8080 make e2e-live-etherscan-proxy-multi-key-rate-limit' >&2; \
		exit 1; \
	fi
	@if [ "$${ATHENA_E2E_ETHERSCAN_PROXY_MULTI_KEY_PROBE:-}" != "1" ]; then \
		printf '%s\n' 'Refusing to intentionally run the Etherscan proxy multi-key probe without confirmation.' >&2; \
		printf '%s\n' 'Run: E2E_LIVE=1 ATHENA_E2E_ETHERSCAN_PROXY_MULTI_KEY_PROBE=1 ATHENA_E2E_ETHERSCAN_API_KEYS=key1,key2 ATHENA_E2E_ETHERSCAN_PROXY_URLS=http://proxy1:8080 make e2e-live-etherscan-proxy-multi-key-rate-limit' >&2; \
		exit 1; \
	fi
	go test $(E2E_GO_TEST_FLAGS) ./e2e/tests/live/ethereumapi -run TestEtherscanProxyMultiKeyAggregateRateLimitProbe -v

.PHONY: serve-docs-local
serve-docs-local:
	mkdocs serve

.PHONY: build-docs
build-docs:
	$(DOCKER) run ${MKDOCS_RUN_ARGS} --rm -it -v ${CURRENT_DIR}:/docs -w /docs --entrypoint "" ${MKDOCS_DOCKER_IMAGE} sh -c 'pip install -r docs/requirements.txt; mkdocs build'

.PHONY: prod-build-local
prod-build-local:
	DOCKER_BUILDKIT=1 $(DOCKER) build --platform=$(TARGET_ARCH) -t $(PROD_IMAGE) .

# use http://127.0.0.1:8080
.PHONY: prod-start-local
prod-start-local:
	$(DOCKER) volume create $(PROD_POSTGRES_VOLUME) >/dev/null
	$(PROD_COMPOSE_LOCAL) up -d postgres
	$(PROD_COMPOSE_LOCAL) --profile tools run --rm athena-migrate athena up --module $(PROD_MIGRATE_MODULE)
	ATHENA_SERVER_DISABLE_AUTH=false $(PROD_COMPOSE_LOCAL) up -d

.PHONY: prod-stop-local
prod-stop-local:
	$(PROD_COMPOSE_LOCAL) down --remove-orphans
	@if $(DOCKER) volume inspect $(PROD_POSTGRES_VOLUME) >/dev/null 2>&1; then \
		$(DOCKER) volume rm $(PROD_POSTGRES_VOLUME); \
	fi

.PHONY: prod-logs-local
prod-logs-local:
	$(PROD_COMPOSE_LOCAL) logs -f $(PROD_LOG_SERVICE)

# ssh -L 8080:127.0.0.1:8080 root@47.245.181.189
# use http://127.0.0.1:8080
.PHONY: prod-deploy-remote
prod-deploy-remote: prod-reset-secrets prod-build-local
	PROD_IMAGE=$(PROD_IMAGE) PROD_COMPOSE_FILE=$(PROD_COMPOSE_FILE) PROD_ENV_FILE=$(PROD_ENV_FILE) REMOTE_APP_DIR=$(REMOTE_APP_DIR) PROD_POSTGRES_VOLUME=$(PROD_POSTGRES_VOLUME) PROD_MIGRATE_MODULE=$(PROD_MIGRATE_MODULE) bash ./hack/prod-remote-deploy.sh deploy

.PHONY: prod-hot-deploy-remote
prod-hot-deploy-remote: prod-build-local
	PROD_IMAGE=$(PROD_IMAGE) PROD_COMPOSE_FILE=$(PROD_COMPOSE_FILE) PROD_ENV_FILE=$(PROD_ENV_FILE) REMOTE_APP_DIR=$(REMOTE_APP_DIR) PROD_POSTGRES_VOLUME=$(PROD_POSTGRES_VOLUME) PROD_MIGRATE_MODULE=$(PROD_MIGRATE_MODULE) bash ./hack/prod-remote-deploy.sh hot-deploy

.PHONY: prod-destroy-remote
prod-destroy-remote:
	PROD_IMAGE=$(PROD_IMAGE) PROD_ENV_FILE=$(PROD_ENV_FILE) REMOTE_APP_DIR=$(REMOTE_APP_DIR) PROD_POSTGRES_VOLUME=$(PROD_POSTGRES_VOLUME) bash ./hack/prod-remote-deploy.sh destroy

.PHONY: cm
cm:
	git add .
	git commit -m "commit"
