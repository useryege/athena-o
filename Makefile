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
MINIO_IMAGE?=athena-minio:9e49d5e7a648
MINIO_MC_IMAGE?=athena-minio-mc:7394ce0dd2a8
MINIO_SERVER_DOCKERFILE?=deploy/minio/Dockerfile.server
MINIO_MC_DOCKERFILE?=deploy/minio/Dockerfile.mc
PROD_COMPOSE_LOCAL=ATHENA_COMPOSE_ENV_FILE=$(PROD_ENV_FILE) ATHENA_SERVER_CONTAINER_USER=$(shell id -u):0 PROD_IMAGE=$(PROD_IMAGE) PROD_POSTGRES_VOLUME=$(PROD_POSTGRES_VOLUME) PROD_REDIS_VOLUME=$(PROD_REDIS_VOLUME) PROD_MINIO_VOLUME=$(PROD_MINIO_VOLUME) MINIO_IMAGE=$(MINIO_IMAGE) MINIO_MC_IMAGE=$(MINIO_MC_IMAGE) $(DOCKER) compose -f $(PROD_COMPOSE_FILE) --env-file $(PROD_ENV_FILE)
BSC_INDEXER_IMAGE?=athena-bsc-transaction-indexer:local
BSC_INDEXER_DOCKERFILE?=deploy/bsc-transaction-indexer/Dockerfile
BSC_INDEXER_COMPOSE_FILE?=deploy/bsc-transaction-indexer/docker-compose.yml
BSC_INDEXER_ENV_FILE?=.env.bsc-transaction-indexer
BSC_INDEXER_REMOTE_APP_DIR?=/opt/athena-bsc-transaction-indexer
BSC_SWAP_INDEXER_IMAGE?=athena-bsc-swap-indexer:local
BSC_SWAP_INDEXER_DOCKERFILE?=deploy/bsc-swap-indexer/Dockerfile
BSC_SWAP_INDEXER_COMPOSE_FILE?=deploy/bsc-swap-indexer/docker-compose.yml
BSC_SWAP_INDEXER_ENV_FILE?=.env.bsc-swap-indexer
BSC_SWAP_INDEXER_REMOTE_APP_DIR?=/opt/athena-bsc-swap-indexer
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

.PHONY: wallet-private-key-ciphertext
wallet-private-key-ciphertext:
	@go run tools/wallet-private-key-ciphertext/main.go $(ARGS)

.PHONY: prod-reset-secrets
prod-reset-secrets:
	@go run tools/prod-env-reset/main.go -env-file $(PROD_ENV_FILE)

.PHONY: deploy-etherscan-gateway-vps
deploy-etherscan-gateway-vps:
	bash ./hack/deploy-etherscan-gateway.sh

.PHONY: deploy-bsc-transaction-indexer-vps
deploy-bsc-transaction-indexer-vps:
	REMOTE_HOST=$(REMOTE_HOST) REMOTE_USER=$(REMOTE_USER) TARGET_ARCH=$(TARGET_ARCH) BSC_INDEXER_IMAGE=$(BSC_INDEXER_IMAGE) BSC_INDEXER_DOCKERFILE=$(BSC_INDEXER_DOCKERFILE) BSC_INDEXER_COMPOSE_FILE=$(BSC_INDEXER_COMPOSE_FILE) BSC_INDEXER_ENV_FILE=$(BSC_INDEXER_ENV_FILE) BSC_INDEXER_REMOTE_APP_DIR=$(BSC_INDEXER_REMOTE_APP_DIR) bash ./hack/deploy-bsc-transaction-indexer.sh

.PHONY: deploy-bsc-swap-indexer-vps
deploy-bsc-swap-indexer-vps:
	REMOTE_HOST=$(REMOTE_HOST) REMOTE_USER=$(REMOTE_USER) TARGET_ARCH=$(TARGET_ARCH) BSC_SWAP_INDEXER_IMAGE=$(BSC_SWAP_INDEXER_IMAGE) BSC_SWAP_INDEXER_DOCKERFILE=$(BSC_SWAP_INDEXER_DOCKERFILE) BSC_SWAP_INDEXER_COMPOSE_FILE=$(BSC_SWAP_INDEXER_COMPOSE_FILE) BSC_SWAP_INDEXER_ENV_FILE=$(BSC_SWAP_INDEXER_ENV_FILE) BSC_SWAP_INDEXER_REMOTE_APP_DIR=$(BSC_SWAP_INDEXER_REMOTE_APP_DIR) bash ./hack/deploy-bsc-swap-indexer.sh

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

.PHONY: athena-bsc-transaction-indexer
athena-bsc-transaction-indexer: clean-debug
	@mkdir -p ${DIST_DIR}
	CGO_ENABLED=0 GOOS=${GOOS} GOARCH=${GOARCH} GODEBUG="tarinsecurepath=0,zipinsecurepath=0" go build -trimpath -v -ldflags '${LDFLAGS} -s -w' -o ${DIST_DIR}/athena-bsc-transaction-indexer ./cmd/athena-bsc-transaction-indexer

.PHONY: bsc-transaction-indexer-build-image
bsc-transaction-indexer-build-image:
	DOCKER_BUILDKIT=1 $(DOCKER) build --platform=$(TARGET_ARCH) -f $(BSC_INDEXER_DOCKERFILE) -t $(BSC_INDEXER_IMAGE) --build-arg GIT_COMMIT=$(GIT_COMMIT) --build-arg GIT_TREE_STATE=$(GIT_TREE_STATE) --build-arg GIT_TAG=$(GIT_TAG) --build-arg BUILD_DATE=$(BUILD_DATE) .

.PHONY: athena-bsc-swap-indexer
athena-bsc-swap-indexer: clean-debug
	@mkdir -p ${DIST_DIR}
	CGO_ENABLED=0 GOOS=${GOOS} GOARCH=${GOARCH} GODEBUG="tarinsecurepath=0,zipinsecurepath=0" go build -trimpath -v -ldflags '${LDFLAGS} -s -w' -o ${DIST_DIR}/athena-bsc-swap-indexer ./cmd/athena-bsc-swap-indexer

.PHONY: bsc-swap-indexer-build-image
bsc-swap-indexer-build-image:
	DOCKER_BUILDKIT=1 $(DOCKER) build --platform=$(TARGET_ARCH) -f $(BSC_SWAP_INDEXER_DOCKERFILE) -t $(BSC_SWAP_INDEXER_IMAGE) --build-arg GIT_COMMIT=$(GIT_COMMIT) --build-arg GIT_TREE_STATE=$(GIT_TREE_STATE) --build-arg GIT_TAG=$(GIT_TAG) --build-arg BUILD_DATE=$(BUILD_DATE) .

# Manage the foreground local development runtime and its resources.
.PHONY: run
run:
	bash ./hack/local-runtime.sh start

.PHONY: stop
stop:
	bash ./hack/local-runtime.sh stop

.PHONY: run-reset
run-reset:
	bash ./hack/local-runtime.sh reset

.PHONY: e2e
e2e:
	go test $(E2E_GO_TEST_FLAGS) $(E2E_PACKAGES)

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
	DOCKER_BUILDKIT=1 $(DOCKER) build --platform=$(TARGET_ARCH) -t $(PROD_IMAGE) .

# use http://127.0.0.1:8080
.PHONY: prod-start-local
prod-start-local:
	$(DOCKER) volume create $(PROD_POSTGRES_VOLUME) >/dev/null
	$(DOCKER) volume create $(PROD_REDIS_VOLUME) >/dev/null
	$(DOCKER) volume create $(PROD_MINIO_VOLUME) >/dev/null
	$(PROD_COMPOSE_LOCAL) up -d postgres
	$(PROD_COMPOSE_LOCAL) --profile tools run --rm athena-migrate athena up --module $(PROD_MIGRATE_MODULE)
	ATHENA_SERVER_DISABLE_AUTH=false $(PROD_COMPOSE_LOCAL) up -d

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
