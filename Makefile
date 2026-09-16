# Freeze command-line/environment inputs before any $(shell ...) can export them.
# $(value) returns literal data; := prevents a later recursive Make expansion.
override SERVICE := $(value SERVICE)
override SERVICES := $(value SERVICES)
override INSTANCE := $(value INSTANCE)
override DB_MODE := $(value DB_MODE)
override ENV_FILE := $(value ENV_FILE)
ifneq ($(origin TRADER_SYNC_IMAGE),undefined)
override TRADER_SYNC_IMAGE := $(value TRADER_SYNC_IMAGE)
endif
ifneq ($(origin ACCOUNT_STATE_MAINTENANCE),undefined)
override ACCOUNT_STATE_MAINTENANCE := $(value ACCOUNT_STATE_MAINTENANCE)
endif
ifneq ($(origin ACCOUNT_STATE_EXTERNAL_CONSUMERS_STOPPED),undefined)
override ACCOUNT_STATE_EXTERNAL_CONSUMERS_STOPPED := $(value ACCOUNT_STATE_EXTERNAL_CONSUMERS_STOPPED)
endif

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
E2E_ENV_FILE ?= .env
E2E_PACKAGES ?= $(shell go list ./e2e/tests/... 2>/dev/null | grep -v '/live/')
E2E_LIVE_PACKAGES ?= ./e2e/tests/live/...

VERSION=$(shell cat ${CURRENT_DIR}/VERSION)
BUILD_DATE:=$(if $(BUILD_DATE),$(BUILD_DATE),$(shell date -u +'%Y-%m-%dT%H:%M:%SZ'))
GIT_COMMIT:=$(if $(GIT_COMMIT),$(GIT_COMMIT),$(shell git rev-parse HEAD))
GIT_TAG:=$(if $(GIT_TAG),$(GIT_TAG),$(shell if git rev-parse --is-inside-work-tree >/dev/null 2>&1 && [ -z "`git status --porcelain`" ]; then git describe --exact-match --tags HEAD 2>/dev/null; fi))
GIT_TREE_STATE:=$(if $(GIT_TREE_STATE),$(GIT_TREE_STATE),$(shell if [ -z "`git status --porcelain`" ]; then echo "clean" ; else echo "dirty"; fi))

# Docker command to use
DOCKER ?= docker
ifneq ($(DOCKER),docker)
$(error Only Docker is supported. Please run make with DOCKER=docker)
endif

PATH:=$(PATH):$(PWD)/hack

PROD_IMAGE?=athena:local
TRADER_SYNC_IMAGE?=athena-trader-sync:local
PROD_COMPOSE_FILE?=docker-compose.prod.yml
PROD_ENV_FILE?=.env.prod
REMOTE_APP_DIR?=/root/athena
REMOTE_USER?=root
IP_GENERATOR_REMOTE_PATH?=/root/ip-generator
PROD_LOG_SERVICE?=
PROD_MIGRATE_MODULE?=all
PROD_POSTGRES_VOLUME?=athena-prod-postgres-data
PROD_REDIS_VOLUME?=athena-prod-redis-data
PROD_MINIO_VOLUME?=athena-prod-minio-data
MINIO_IMAGE?=athena-minio:9e49d5e7a648-go1.27.1
MINIO_MC_IMAGE?=athena-minio-mc:7394ce0dd2a8-go1.27.1
MINIO_SERVER_DOCKERFILE?=deploy/minio/Dockerfile.server
MINIO_MC_DOCKERFILE?=deploy/minio/Dockerfile.mc
PROD_COMPOSE_LOCAL=ATHENA_COMPOSE_ENV_FILE=$(PROD_ENV_FILE) ATHENA_SERVER_CONTAINER_USER=$(shell id -u):0 PROD_IMAGE=$(PROD_IMAGE) PROD_POSTGRES_VOLUME=$(PROD_POSTGRES_VOLUME) PROD_REDIS_VOLUME=$(PROD_REDIS_VOLUME) PROD_MINIO_VOLUME=$(PROD_MINIO_VOLUME) MINIO_IMAGE=$(MINIO_IMAGE) MINIO_MC_IMAGE=$(MINIO_MC_IMAGE) $(DOCKER) compose -f $(PROD_COMPOSE_FILE) --env-file $(PROD_ENV_FILE)
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

.PHONY: jwt-secret
jwt-secret:
	@go run tools/jwt-secret/main.go

.PHONY: service-password
service-password:
	@go run tools/service-password/main.go

.PHONY: notify-task-complete
notify-task-complete:
	@go run tools/task-completion-email/main.go

.PHONY: prod-reset-secrets
prod-reset-secrets:
	@go run tools/prod-env-reset/main.go -env-file $(PROD_ENV_FILE)

.PHONY: deploy-etherscan-gateway-vps
deploy-etherscan-gateway-vps:
	bash ./hack/deploy-etherscan-gateway.sh

.PHONY: install-docker-vps
install-docker-vps:
	REMOTE_HOST=$(REMOTE_HOST) REMOTE_USER=$(REMOTE_USER) bash ./hack/install-docker-vps.sh

.PHONY: deploy-ip-generator
deploy-ip-generator:
	REMOTE_HOST=$(REMOTE_HOST) REMOTE_USER=$(REMOTE_USER) IP_GENERATOR_REMOTE_PATH=$(IP_GENERATOR_REMOTE_PATH) bash ./hack/deploy-ip-generator.sh

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

.PHONY: athena-etherscan-gateway
athena-etherscan-gateway: clean-debug
	CGO_ENABLED=${CGO_FLAG} GOOS=${GOOS} GOARCH=${GOARCH} GODEBUG="tarinsecurepath=0,zipinsecurepath=0" go build -trimpath -v -ldflags '${LDFLAGS} -s -w' -o ${DIST_DIR}/athena-etherscan-gateway ./cmd/athena-etherscan-gateway

# Manage the foreground local development runtime and its resources.
.PHONY: run
run:
	@exec bash ./hack/local-runtime.sh start

.PHONY: stop
stop:
	@exec bash ./hack/local-runtime.sh stop

.PHONY: solana-discovery-build solana-discovery-run solana-discovery-stop
solana-discovery-build:
	mkdir -p .tmp/bin
	go build -o .tmp/bin/athena-solana-discovery ./cmd/athena-solana-discovery

solana-discovery-run:
	bash ./hack/solana-local.sh start solana-discovery

solana-discovery-stop:
	bash ./hack/solana-local.sh stop solana-discovery

.PHONY: run-reset
run-reset:
	@exec bash ./hack/local-runtime.sh reset

.PHONY: e2e
e2e:
	go test $(E2E_GO_TEST_FLAGS) $(E2E_PACKAGES)

.PHONY: ui-acceptance
ui-acceptance:
	@bash ./hack/ui-acceptance.sh

.PHONY: test-dom test-visual test-fuzz test-ai-tools
test-dom:
	cd ui && node node_modules/yarn/bin/yarn.js test:dom

test-visual:
	cd ui && node node_modules/yarn/bin/yarn.js test:visual

test-fuzz:
	GOPROXY=off GONOPROXY=none GOSUMDB=off GOTOOLCHAIN=local go test -mod=readonly ./internal/tradersync -run='^$$' -fuzz='^FuzzCursorRoundTrip$$' -fuzztime=10s -parallel=2 -count=1

test-ai-tools:
	+$(MAKE) test-dom
	+$(MAKE) test-visual
	+$(MAKE) test-fuzz

.PHONY: install-ai-dev-tools ai-dev-tools-check lint-shell vuln-check ui-a11y
install-ai-dev-tools:
	@bash ./hack/ai-dev-tools.sh install

ai-dev-tools-check:
	@bash ./hack/ai-dev-tools.sh check

lint-shell:
	@bash ./hack/ai-dev-tools.sh lint-shell

vuln-check:
	@bash ./hack/ai-dev-tools.sh vuln-check

ui-a11y:
	@UI_ACCEPTANCE_MODE=isolated UI_ACCEPTANCE_SUITE=a11y bash ./hack/ui-acceptance.sh

.PHONY: e2e-etherscan-manager
e2e-etherscan-manager:
	go test $(E2E_GO_TEST_FLAGS) ./e2e/tests/etherscanmanager

.PHONY: e2e-bsc-block-trace
e2e-bsc-block-trace:
	go run ./e2e/cmd/bsc-block-trace $(if $(BLOCK_NUMBER),-block "$(BLOCK_NUMBER)",)

.PHONY: e2e-live
e2e-live:
	@if [ "$${E2E_LIVE:-}" != "1" ]; then \
		printf '%s\n' 'Refusing to run live E2E tests without explicit confirmation.' >&2; \
		printf '%s\n' 'Run: E2E_LIVE=1 make e2e-live' >&2; \
		exit 1; \
	fi
	go test $(E2E_GO_TEST_FLAGS) $(E2E_LIVE_PACKAGES)

.PHONY: e2e-live-etherscan-manager
e2e-live-etherscan-manager:
	@if [ "$${E2E_LIVE:-}" != "1" ]; then \
		printf '%s\n' 'Refusing to run etherscan-manager live E2E tests without explicit confirmation.' >&2; \
		printf '%s\n' 'Run: E2E_LIVE=1 make e2e-live-etherscan-manager' >&2; \
		exit 1; \
	fi
	go test $(E2E_GO_TEST_FLAGS) ./e2e/tests/live/etherscanmanager

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
	go test $(E2E_GO_TEST_FLAGS) ./e2e/tests/live/etherscanapi -run TestEtherscanFreePlanRateLimitProbe -v

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
	go test $(E2E_GO_TEST_FLAGS) ./e2e/tests/live/etherscanapi -run TestEtherscanMultiKeyAggregateRateLimitProbe -v

.PHONY: e2e-live-etherscan-multi-key-staggered-rate-limit
e2e-live-etherscan-multi-key-staggered-rate-limit:
	@if [ "$${E2E_LIVE:-}" != "1" ]; then \
		printf '%s\n' 'Refusing to run staggered Etherscan multi-key probe without live test confirmation.' >&2; \
		printf '%s\n' 'Run: E2E_LIVE=1 ATHENA_E2E_ETHERSCAN_MULTI_KEY_STAGGERED_PROBE=1 ATHENA_E2E_ETHERSCAN_API_KEYS=key1,key2,key3 make e2e-live-etherscan-multi-key-staggered-rate-limit' >&2; \
		exit 1; \
	fi
	@if [ "$${ATHENA_E2E_ETHERSCAN_MULTI_KEY_STAGGERED_PROBE:-}" != "1" ]; then \
		printf '%s\n' 'Refusing to intentionally run the staggered Etherscan multi-key probe without confirmation.' >&2; \
		printf '%s\n' 'Run: E2E_LIVE=1 ATHENA_E2E_ETHERSCAN_MULTI_KEY_STAGGERED_PROBE=1 ATHENA_E2E_ETHERSCAN_API_KEYS=key1,key2,key3 make e2e-live-etherscan-multi-key-staggered-rate-limit' >&2; \
		exit 1; \
	fi
	go test $(E2E_GO_TEST_FLAGS) ./e2e/tests/live/etherscanapi -run TestEtherscanMultiKeyAggregateStaggeredRateLimitProbe -v

.PHONY: e2e-live-etherscan-gateway-multi-key-staggered-success
e2e-live-etherscan-gateway-multi-key-staggered-success:
	@env_file="$(E2E_ENV_FILE)"; \
	if [ ! -f "$$env_file" ]; then \
		printf '%s\n' 'Refusing to run staggered Etherscan Gateway multi-key probe without an env file.' >&2; \
		printf '%s\n' 'Run: E2E_ENV_FILE=.env E2E_LIVE=1 ATHENA_E2E_ETHERSCAN_GATEWAY_MULTI_KEY_STAGGERED_PROBE=1 make e2e-live-etherscan-gateway-multi-key-staggered-success' >&2; \
		exit 1; \
	fi
	@env_file="$(E2E_ENV_FILE)"; \
	case "$$env_file" in /*|*/*) env_source="$$env_file" ;; *) env_source="./$$env_file" ;; esac; \
	set -a; . "$$env_source"; set +a; \
	if [ "$${E2E_LIVE:-}" != "1" ]; then \
		printf '%s\n' 'Refusing to run staggered Etherscan Gateway multi-key probe without live test confirmation.' >&2; \
		printf '%s\n' 'Run: E2E_LIVE=1 ATHENA_E2E_ETHERSCAN_GATEWAY_MULTI_KEY_STAGGERED_PROBE=1 make e2e-live-etherscan-gateway-multi-key-staggered-success' >&2; \
		exit 1; \
	fi; \
	if [ "$${ATHENA_E2E_ETHERSCAN_GATEWAY_MULTI_KEY_STAGGERED_PROBE:-}" != "1" ]; then \
		printf '%s\n' 'Refusing to intentionally run the staggered Etherscan Gateway multi-key probe without confirmation.' >&2; \
		printf '%s\n' 'Run: E2E_LIVE=1 ATHENA_E2E_ETHERSCAN_GATEWAY_MULTI_KEY_STAGGERED_PROBE=1 make e2e-live-etherscan-gateway-multi-key-staggered-success' >&2; \
		exit 1; \
	fi; \
	if [ -z "$${ATHENA_E2E_ETHERSCAN_API_KEYS:-}" ]; then \
		printf '%s\n' 'Refusing to run staggered Etherscan Gateway multi-key probe without ATHENA_E2E_ETHERSCAN_API_KEYS in the env file.' >&2; \
		exit 1; \
	fi; \
	if [ -z "$${ATHENA_ETHERSCAN_GATEWAY_AUTH_TOKEN:-}" ]; then \
		printf '%s\n' 'Refusing to run staggered Etherscan Gateway multi-key probe without ATHENA_ETHERSCAN_GATEWAY_AUTH_TOKEN in the env file.' >&2; \
		exit 1; \
	fi; \
	if [ -z "$${ATHENA_E2E_ETHERSCAN_GATEWAY_ADDRS:-}" ] && [ -z "$${ETHERSCAN_GATEWAY_IPS:-}" ]; then \
		printf '%s\n' 'Refusing to run staggered Etherscan Gateway multi-key probe without ATHENA_E2E_ETHERSCAN_GATEWAY_ADDRS or ETHERSCAN_GATEWAY_IPS in the env file.' >&2; \
		exit 1; \
	fi; \
	go test $(E2E_GO_TEST_FLAGS) ./e2e/tests/live/etherscangateway -run TestEtherscanGatewayMultiKeyStaggeredSuccessProbe -v

.PHONY: minio-images-local
minio-images-local:
	DOCKER_BUILDKIT=1 $(DOCKER) build --platform=$(TARGET_ARCH) -f $(MINIO_SERVER_DOCKERFILE) -t $(MINIO_IMAGE) deploy/minio
	DOCKER_BUILDKIT=1 $(DOCKER) build --platform=$(TARGET_ARCH) -f $(MINIO_MC_DOCKERFILE) -t $(MINIO_MC_IMAGE) deploy/minio

.PHONY: prod-build-local
prod-build-local: minio-images-local
	$(MAKE) build-service-image SERVICE=trader-sync
	DOCKER_BUILDKIT=1 $(DOCKER) build --platform=$(TARGET_ARCH) -t $(PROD_IMAGE) .

# use http://127.0.0.1:8080
.PHONY: prod-start-local
prod-start-local:
	ATHENA_SERVER_CONTAINER_USER=$$(id -u):0 bash ./hack/prod-start-local.sh

.PHONY: prod-stop-local
prod-stop-local:
	$(PROD_COMPOSE_LOCAL) down --remove-orphans
	@if $(DOCKER) volume inspect $(PROD_POSTGRES_VOLUME) >/dev/null 2>&1; then \
		$(DOCKER) volume rm $(PROD_POSTGRES_VOLUME); \
	fi
	@if $(DOCKER) volume inspect $(PROD_REDIS_VOLUME) >/dev/null 2>&1; then \
		$(DOCKER) volume rm $(PROD_REDIS_VOLUME); \
	fi
	@if $(DOCKER) volume inspect $(PROD_MINIO_VOLUME) >/dev/null 2>&1; then \
		$(DOCKER) volume rm $(PROD_MINIO_VOLUME); \
	fi

.PHONY: prod-logs-local
prod-logs-local:
	$(PROD_COMPOSE_LOCAL) logs -f $(PROD_LOG_SERVICE)

# ssh -L 8080:127.0.0.1:8080 root@47.245.181.189
# use http://127.0.0.1:8080
.PHONY: prod-deploy-remote
prod-deploy-remote: prod-reset-secrets prod-build-local
	PROD_IMAGE=$(PROD_IMAGE) MINIO_IMAGE=$(MINIO_IMAGE) MINIO_MC_IMAGE=$(MINIO_MC_IMAGE) PROD_COMPOSE_FILE=$(PROD_COMPOSE_FILE) PROD_ENV_FILE=$(PROD_ENV_FILE) REMOTE_APP_DIR=$(REMOTE_APP_DIR) PROD_POSTGRES_VOLUME=$(PROD_POSTGRES_VOLUME) PROD_REDIS_VOLUME=$(PROD_REDIS_VOLUME) PROD_MINIO_VOLUME=$(PROD_MINIO_VOLUME) PROD_MIGRATE_MODULE=$(PROD_MIGRATE_MODULE) bash ./hack/prod-remote-deploy.sh deploy

.PHONY: prod-hot-deploy-remote
prod-hot-deploy-remote: prod-build-local
	PROD_IMAGE=$(PROD_IMAGE) MINIO_IMAGE=$(MINIO_IMAGE) MINIO_MC_IMAGE=$(MINIO_MC_IMAGE) PROD_COMPOSE_FILE=$(PROD_COMPOSE_FILE) PROD_ENV_FILE=$(PROD_ENV_FILE) REMOTE_APP_DIR=$(REMOTE_APP_DIR) PROD_POSTGRES_VOLUME=$(PROD_POSTGRES_VOLUME) PROD_REDIS_VOLUME=$(PROD_REDIS_VOLUME) PROD_MINIO_VOLUME=$(PROD_MINIO_VOLUME) PROD_MIGRATE_MODULE=$(PROD_MIGRATE_MODULE) bash ./hack/prod-remote-deploy.sh hot-deploy

.PHONY: prod-destroy-remote
prod-destroy-remote:
	PROD_IMAGE=$(PROD_IMAGE) MINIO_IMAGE=$(MINIO_IMAGE) MINIO_MC_IMAGE=$(MINIO_MC_IMAGE) PROD_ENV_FILE=$(PROD_ENV_FILE) REMOTE_APP_DIR=$(REMOTE_APP_DIR) PROD_POSTGRES_VOLUME=$(PROD_POSTGRES_VOLUME) PROD_REDIS_VOLUME=$(PROD_REDIS_VOLUME) PROD_MINIO_VOLUME=$(PROD_MINIO_VOLUME) bash ./hack/prod-remote-deploy.sh destroy

.PHONY: cm
cm:
	git add .
	git commit -m "commit"

.PHONY: account-state-migrate-build account-state-schema-contract
account-state-migrate-build:
	CGO_ENABLED=$(CGO_FLAG) go build -o $(DIST_DIR)/athena-account-state-migrate ./cmd/athena-account-state-migrate

# Requires an explicit test administrator DSN; the tool creates and removes only its own random database.
account-state-schema-contract:
	go run ./tools/account-state-schema-contract > internal/accountstate/schema/contract.json.tmp
	mv internal/accountstate/schema/contract.json.tmp internal/accountstate/schema/contract.json

# Values are exported as data, never interpolated into a shell recipe.
export SERVICE SERVICES INSTANCE DB_MODE ENV_FILE ACCOUNT_STATE_MAINTENANCE ACCOUNT_STATE_EXTERNAL_CONSUMERS_STOPPED
.PHONY: build-service run-service run-services runtime-status stop-instance reset-instance seed-service account-state-migrate
build-service:
	@exec bash ./hack/run-local-runtime.sh make-build
run-service:
	@exec bash ./hack/run-local-runtime.sh make-run-service
run-services:
	@exec bash ./hack/run-local-runtime.sh make-run-services
runtime-status:
	@exec bash ./hack/run-local-runtime.sh make-status
stop-instance:
	@exec bash ./hack/run-local-runtime.sh make-stop
reset-instance:
	@exec bash ./hack/run-local-runtime.sh make-reset
seed-service:
	@exec bash ./hack/run-local-runtime.sh make-seed
account-state-migrate:
	go run ./cmd/athena-account-state-migrate up --timeout=120s
	go run ./cmd/athena-account-state-migrate verify --timeout=120s

# Export user input as data; do not splice image tags or service names into shell code.
export TRADER_SYNC_IMAGE SERVICE TARGET_ARCH VERSION GIT_COMMIT GIT_TREE_STATE GIT_TAG BUILD_DATE
export PROD_IMAGE PROD_COMPOSE_FILE PROD_ENV_FILE REMOTE_APP_DIR REMOTE_USER
export PROD_POSTGRES_VOLUME PROD_REDIS_VOLUME PROD_MINIO_VOLUME PROD_MIGRATE_MODULE
export MINIO_IMAGE MINIO_MC_IMAGE

.PHONY: build-service-image prod-trader-sync-deploy-remote
build-service-image:
	@test "$$SERVICE" = trader-sync || { echo 'build-service-image requires SERVICE=trader-sync' >&2; exit 1; }
	@DOCKER_BUILDKIT=1 docker build --platform="$$TARGET_ARCH" -f deploy/trader-sync/Dockerfile -t "$$TRADER_SYNC_IMAGE" \
	  --build-arg "VERSION=$$VERSION" --build-arg "GIT_COMMIT=$$GIT_COMMIT" --build-arg "GIT_TREE_STATE=$$GIT_TREE_STATE" \
	  --build-arg "GIT_TAG=$$GIT_TAG" --build-arg "BUILD_DATE=$$BUILD_DATE" .

prod-trader-sync-deploy-remote:
	$(MAKE) build-service-image SERVICE=trader-sync
	bash ./hack/prod-remote-deploy.sh trader-sync-deploy

.PHONY: trader-sync-acceptance
trader-sync-acceptance:
	@bash ./hack/trader-sync-independent-acceptance.sh
