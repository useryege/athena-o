PACKAGE=github.com/useryege/athena/common
CURRENT_DIR=$(shell pwd)
DIST_DIR=${CURRENT_DIR}/dist
CLI_NAME=athena
BIN_NAME=athena

UNAME_S:=$(shell uname)
IS_DARWIN:=$(if $(filter Darwin, $(UNAME_S)),true,false)

# When using OSX/Darwin, you might need to enable CGO for local builds
DEFAULT_CGO_FLAG:=0
ifeq ($(IS_DARWIN),true)
    DEFAULT_CGO_FLAG:=1
endif
CGO_FLAG?=${DEFAULT_CGO_FLAG}

GEN_RESOURCES_CLI_NAME=athena-resources-gen

HOST_OS:=$(shell go env GOOS)
HOST_ARCH:=$(shell go env GOARCH)

TARGET_ARCH?=linux/amd64

VERSION=$(shell cat ${CURRENT_DIR}/VERSION)
BUILD_DATE:=$(if $(BUILD_DATE),$(BUILD_DATE),$(shell date -u +'%Y-%m-%dT%H:%M:%SZ'))
GIT_COMMIT:=$(if $(GIT_COMMIT),$(GIT_COMMIT),$(shell git rev-parse HEAD))
GIT_TAG:=$(if $(GIT_TAG),$(GIT_TAG),$(shell if [ -z "`git status --porcelain`" ]; then git describe --exact-match --tags HEAD 2>/dev/null; fi))
GIT_TREE_STATE:=$(if $(GIT_TREE_STATE),$(GIT_TREE_STATE),$(shell if [ -z "`git status --porcelain`" ]; then echo "clean" ; else echo "dirty"; fi))
VOLUME_MOUNT=$(shell if test "$(go env GOOS)" = "darwin"; then echo ":delegated"; elif test selinuxenabled; then echo ":delegated"; else echo ""; fi)
KUBECTL_VERSION=$(shell go list -m k8s.io/client-go | head -n 1 | rev | cut -d' ' -f1 | rev)

GOPATH?=$(shell if test -x `which go`; then go env GOPATH; else echo "$(HOME)/go"; fi)
GOCACHE?=$(HOME)/.cache/go-build

# Docker command to use
DOCKER ?= docker
ifneq ($(DOCKER),docker)
$(error Only Docker is supported. Please run make with DOCKER=docker)
endif

DOCKER_SRCDIR ?= $(GOPATH)/src
DOCKER_WORKDIR ?= /go/src/github.com/useryege/athena
# Allows you to control which Docker network the test-util containers attach to.
# This is particularly useful if you are running Kubernetes in Docker (e.g., k3d)
# and want the test containers to reach the Kubernetes API via an already-existing Docker network.
DOCKER_NETWORK ?= default

ifneq ($(DOCKER_NETWORK),default)
DOCKER_NETWORK_ARG := --network $(DOCKER_NETWORK)
else
DOCKER_NETWORK_ARG :=
endif

ATHENA_PROCFILE?=Procfile

# pointing to python 3.12 
MKDOCS_DOCKER_IMAGE?=python:3.12-alpine
MKDOCS_RUN_ARGS?=

# Configuration for building athena-test-tools image
TEST_TOOLS_NAMESPACE ?=
TEST_TOOLS_IMAGE = athena-test-tools
TEST_TOOLS_TAG ?= latest
ifdef TEST_TOOLS_NAMESPACE
TEST_TOOLS_PREFIX = $(TEST_TOOLS_NAMESPACE)/
endif

# You can change the ports where Athena components will be listening on by
# setting the appropriate environment variables before running make.
ATHENA_E2E_APISERVER_PORT?=8080
ATHENA_E2E_REPOSERVER_PORT?=8081
ATHENA_E2E_REDIS_PORT?=6379
ATHENA_E2E_DEX_PORT?=5556
ATHENA_E2E_YARN_HOST?=localhost
ATHENA_E2E_DISABLE_AUTH?=
ATHENA_E2E_DIR?=/tmp/athena-e2e

ATHENA_E2E_TEST_TIMEOUT?=90m
ATHENA_E2E_RERUN_FAILS?=5

ATHENA_IN_CI?=false
ATHENA_TEST_E2E?=true
ATHENA_BIN_MODE?=true

ATHENA_LINT_GOGC?=20

# Depending on where we are (legacy or non-legacy pwd), we need to use
# different Docker volume mounts for our source tree
LEGACY_PATH=$(GOPATH)/src/github.com/useryege/athena
ifeq ("$(PWD)","$(LEGACY_PATH)")
DOCKER_SRC_MOUNT="$(DOCKER_SRCDIR):/go/src$(VOLUME_MOUNT)"
else
DOCKER_SRC_MOUNT="$(PWD):/go/src/github.com/useryege/athena$(VOLUME_MOUNT)"
endif

# User and group IDs to map to the test container
CONTAINER_UID=$(shell id -u)
CONTAINER_GID=$(shell id -g)

# Set SUDO to sudo to run privileged commands with sudo
SUDO?=


.PHONY: print-env-vars
print-env-vars:
	@echo "CURRENT_DIR=$(CURRENT_DIR)"
	@echo "DIST_DIR=$(DIST_DIR)"
	@echo "HOST_OS=$(HOST_OS)"
	@echo "HOST_ARCH=$(HOST_ARCH)"
	@echo "VOLUME_MOUNT=$(VOLUME_MOUNT)"
	@echo "GOPATH=$(GOPATH)"
	@echo "GOCACHE=$(GOCACHE)"
	@echo "DOCKER=$(DOCKER)"
	@echo "DOCKER_SRCDIR=$(DOCKER_SRCDIR)"
	@echo "DOCKER_WORKDIR=$(DOCKER_WORKDIR)"
	@echo "DOCKER_NETWORK=$(DOCKER_NETWORK)"
	@echo "DOCKER_NETWORK_ARG=$(DOCKER_NETWORK_ARG)"
	@echo "TEST_TOOLS_NAMESPACE=$(TEST_TOOLS_NAMESPACE)"
	@echo "TEST_TOOLS_IMAGE=$(TEST_TOOLS_IMAGE)"
	@echo "TEST_TOOLS_TAG=$(TEST_TOOLS_TAG)"
	@echo "TEST_TOOLS_PREFIX=$(TEST_TOOLS_PREFIX)"
	@echo "DOCKER_SRC_MOUNT=$(DOCKER_SRC_MOUNT)"
	@echo "CONTAINER_UID=$(CONTAINER_UID)"
	@echo "CONTAINER_GID=$(CONTAINER_GID)"
	@echo "SUDO=$(SUDO)"
	@echo "ATHENA_LINT_GOGC=$(ATHENA_LINT_GOGC)"

# Runs any command in the athena-test-utils container in server mode
# Server mode container will start with uid 0 and drop privileges during runtime
define run-in-test-server
	$(SUDO) $(DOCKER) run --rm -it \
		--name athena-test-server \
		-u $(CONTAINER_UID):$(CONTAINER_GID) \
		-e USER_ID=$(CONTAINER_UID) \
		-e HOME=/home/user \
		-e GOPATH=/go \
		-e GOCACHE=/tmp/go-build-cache \
		-e ATHENA_IN_CI=$(ATHENA_IN_CI) \
		-e ATHENA_E2E_TEST=$(ATHENA_E2E_TEST) \
		-e ATHENA_E2E_YARN_HOST=$(ATHENA_E2E_YARN_HOST) \
		-e ATHENA_E2E_DISABLE_AUTH=$(ATHENA_E2E_DISABLE_AUTH) \
		-e ATHENA_TLS_DATA_PATH=${ATHENA_TLS_DATA_PATH:-/tmp/athena-local/tls} \
		-e ATHENA_SSH_DATA_PATH=${ATHENA_SSH_DATA_PATH:-/tmp/athena-local/ssh} \
		-e ATHENA_GPG_DATA_PATH=${ATHENA_GPG_DATA_PATH:-/tmp/athena-local/gpg/source} \
		-e ATHENA_APPLICATION_NAMESPACES \
		-e GITHUB_TOKEN \
		-v ${DOCKER_SRC_MOUNT} \
		-v ${GOPATH}/pkg/mod:/go/pkg/mod${VOLUME_MOUNT} \
		-v ${GOCACHE}:/tmp/go-build-cache${VOLUME_MOUNT} \
		-v ${HOME}/.kube:/home/user/.kube${VOLUME_MOUNT} \
		-w ${DOCKER_WORKDIR} \
		-p ${ATHENA_E2E_APISERVER_PORT}:8080 \
		-p 4000:4000 \
		-p 5000:5000 \
		$(DOCKER_NETWORK_ARG)\
		$(PODMAN_ARGS) \
		$(TEST_TOOLS_PREFIX)$(TEST_TOOLS_IMAGE):$(TEST_TOOLS_TAG) \
		bash -c "$(1)"
endef

define run-in-test-client
	$(SUDO) $(DOCKER) run --rm -it \
	  --name athena-test-client \
		-u $(CONTAINER_UID):$(CONTAINER_GID) \
		-e HOME=/home/user \
		-e GOPATH=/go \
		-e ATHENA_E2E_K3S=$(ATHENA_E2E_K3S) \
		-e GITHUB_TOKEN \
		-e GOCACHE=/tmp/go-build-cache \
		-e ATHENA_LINT_GOGC=$(ATHENA_LINT_GOGC) \
		-v $(DOCKER_SRC_MOUNT) \
		-v $(GOPATH)/pkg/mod:/go/pkg/mod$(VOLUME_MOUNT) \
		-v $(GOCACHE):/tmp/go-build-cache$(VOLUME_MOUNT) \
		-v $(HOME)/.kube:/home/user/.kube$(VOLUME_MOUNT) \
		-w $(DOCKER_WORKDIR) \
		$(DOCKER_NETWORK_ARG) \
		$(TEST_TOOLS_PREFIX)$(TEST_TOOLS_IMAGE):$(TEST_TOOLS_TAG) \
		bash -c "$(1)"
endef

#
define exec-in-test-server
	$(SUDO) $(DOCKER) exec -it -u $(CONTAINER_UID):$(CONTAINER_GID) -e ATHENA_E2E_RECORD=$(ATHENA_E2E_RECORD) -e ATHENA_E2E_K3S=$(ATHENA_E2E_K3S) athena-test-server $(1)
endef

PATH:=$(PATH):$(PWD)/hack

# docker image publishing options
DOCKER_PUSH?=false
IMAGE_NAMESPACE?=
# perform static compilation
DEFAULT_STATIC_BUILD:=true
ifeq ($(IS_DARWIN),true)
    DEFAULT_STATIC_BUILD:=false
endif
STATIC_BUILD?=${DEFAULT_STATIC_BUILD}
# build development images
DEV_IMAGE?=false
ATHENA_GPG_ENABLED?=true
ATHENA_E2E_APISERVER_PORT?=8080

ifeq (${COVERAGE_ENABLED}, true)
# We use this in the cli-local target to enable code coverage for e2e tests.
COVERAGE_FLAG=-cover
else
COVERAGE_FLAG=
endif

override LDFLAGS += \
  -X ${PACKAGE}.version=${VERSION} \
  -X ${PACKAGE}.buildDate=${BUILD_DATE} \
  -X ${PACKAGE}.gitCommit=${GIT_COMMIT} \
  -X ${PACKAGE}.gitTreeState=${GIT_TREE_STATE}\
  -X ${PACKAGE}.kubectlVersion=${KUBECTL_VERSION}\
  -X "${PACKAGE}.extraBuildInfo=${EXTRA_BUILD_INFO}"

ifeq (${STATIC_BUILD}, true)
override LDFLAGS += -extldflags "-static"
endif


ifneq (${GIT_TAG},)
IMAGE_TAG=${GIT_TAG}
override LDFLAGS += -X ${PACKAGE}.gitTag=${GIT_TAG}
else
IMAGE_TAG?=latest
endif

ifeq (${DOCKER_PUSH},true)
ifndef IMAGE_NAMESPACE
$(error IMAGE_NAMESPACE must be set to push images (e.g. IMAGE_NAMESPACE=useryege))
endif
endif

ifdef IMAGE_NAMESPACE
IMAGE_PREFIX=${IMAGE_NAMESPACE}/
endif

ifndef IMAGE_REGISTRY
IMAGE_REGISTRY="quay.io"
endif

# Installs all tools required to build and test Athena locally
.PHONY: install-tools-local
install-tools-local: install-test-tools-local install-codegen-tools-local install-go-tools-local

# Installs all tools required for running unit & end-to-end tests (Linux packages)
.PHONY: install-test-tools-local
install-test-tools-local:
	./hack/install.sh kustomize
	./hack/install.sh helm
	./hack/install.sh gotestsum
	./hack/install.sh oras
	./hack/install.sh kind


# Installs all tools required for running codegen (Go packages)
.PHONY: install-go-tools-local
install-go-tools-local:
	./hack/install.sh codegen-go-tools
	./hack/install.sh lint-tools

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

.PHONY: clientgen
clientgen:
	export GO111MODULE=off
	./hack/update-codegen.sh

.PHONY: clidocsgen
clidocsgen:
	go run tools/cmd-docs/main.go

.PHONY: manifests-local
manifests-local:
	./hack/update-manifests.sh

.PHONY: mod-download-local
mod-download-local:
	go mod download && go mod tidy

.PHONY: mod-vendor-local
mod-vendor-local: mod-download-local
	go mod vendor

# new codegen-local
.PHONY: codegen-local
codegen-local: mod-vendor-local mockgen gogen protogen clientgen clidocsgen manifests-local
	rm -rf vendor/

.PHONY: test-tools-image
test-tools-image:
ifndef SKIP_TEST_TOOLS_IMAGE
	$(SUDO) $(DOCKER) build --build-arg UID=$(CONTAINER_UID) -t $(TEST_TOOLS_PREFIX)$(TEST_TOOLS_IMAGE) -f test/container/Dockerfile .
	$(SUDO) $(DOCKER) tag $(TEST_TOOLS_PREFIX)$(TEST_TOOLS_IMAGE) $(TEST_TOOLS_PREFIX)$(TEST_TOOLS_IMAGE):$(TEST_TOOLS_TAG)
endif

.PHONY: codegen
codegen: test-tools-image
	$(call run-in-test-client,make codegen-local)


# Build all Go code
.PHONY: build
build: test-tools-image
	mkdir -p $(GOCACHE)
	$(call run-in-test-client, make build-local)

# Build all Go code (local version)
.PHONY: build-local
build-local:
	GODEBUG="tarinsecurepath=0,zipinsecurepath=0" go build -v `go list ./... | grep -v 'resource_customizations\|test/e2e'`

# Run all unit tests
#
# If TEST_MODULE is set (to fully qualified module name), only this specific
# module will be tested.
.PHONY: test
test: test-tools-image
	mkdir -p $(GOCACHE)
	$(call run-in-test-client,make TEST_MODULE=$(TEST_MODULE) test-local)

# Run all unit tests (local version)
.PHONY: test-local
test-local:
	if test "$(TEST_MODULE)" = ""; then \
		DIST_DIR=${DIST_DIR} RERUN_FAILS=0 PACKAGES=`go list ./... | grep -v 'test/e2e'` ./hack/test.sh -args -test.gocoverdir="$(PWD)/test-results"; \
	else \
		DIST_DIR=${DIST_DIR} RERUN_FAILS=0 PACKAGES="$(TEST_MODULE)" ./hack/test.sh -args -test.gocoverdir="$(PWD)/test-results" "$(TEST_MODULE)"; \
	fi

.PHONY: test-race
test-race: test-tools-image
	mkdir -p $(GOCACHE)
	$(call run-in-test-client,make TEST_MODULE=$(TEST_MODULE) test-race-local)

# Run all unit tests, with data race detection, skipping known failures (local version)
.PHONY: test-race-local
test-race-local:
	if test "$(TEST_MODULE)" = ""; then \
		DIST_DIR=${DIST_DIR} RERUN_FAILS=0 PACKAGES=`go list ./... | grep -v 'test/e2e'` ./hack/test.sh -race -args -test.gocoverdir="$(PWD)/test-results"; \
	else \
		DIST_DIR=${DIST_DIR} RERUN_FAILS=0 PACKAGES="$(TEST_MODULE)" ./hack/test.sh -race -args -test.gocoverdir="$(PWD)/test-results"; \
	fi

# Run linter on the code
.PHONY: lint
lint: test-tools-image
	$(call run-in-test-client,make lint-local)

# Run linter on the code (local version)
.PHONY: lint-local
lint-local:
	golangci-lint --version
	# NOTE: If you get a "Killed" OOM message, try reducing the value of GOGC
	# See https://github.com/golangci/golangci-lint#memory-usage-of-golangci-lint
	GOGC=$(ATHENA_LINT_GOGC) GOMAXPROCS=2 golangci-lint run --fix --verbose

# Verify that kubectl can connect to your K8s cluster from Docker
.PHONY: verify-kube-connect
verify-kube-connect: test-tools-image
	$(call run-in-test-client,kubectl version)


# Runs pre-commit validation with the virtualized toolchain
.PHONY: pre-commit
pre-commit: codegen build lint test


.PHONY: serve-docs
serve-docs:
	$(DOCKER) run ${MKDOCS_RUN_ARGS} --rm -it -p 8000:8000 -v ${CURRENT_DIR}:/docs -w /docs --entrypoint "" ${MKDOCS_DOCKER_IMAGE} sh -c 'pip install -r docs/requirements.txt; mkdocs serve -a $$(ip route get 1 | awk '\''{print $$7}'\''):8000'


.PHONY: start
start: test-tools-image
	$(DOCKER) version
	$(call run-in-test-server,make ATHENA_PROCFILE=test/container/Procfile start-local ATHENA_START=${ATHENA_START})

# Starts a local instance of Athena
.PHONY: start-local
start-local: mod-vendor-local dep-ui-local cli-local
	# check we can connect to Docker to start Redis
	killall goreman || true
	kubectl create ns athena || true
	rm -rf /tmp/athena-local
	mkdir -p /tmp/athena-local
	mkdir -p /tmp/athena-local/gpg/keys && chmod 0700 /tmp/athena-local/gpg/keys
	mkdir -p /tmp/athena-local/gpg/source
	REDIS_PASSWORD=$(shell kubectl get secret athena-redis -o jsonpath='{.data.auth}' | base64 -d) \
	ATHENA_ZJWT_FEATURE_FLAG=always \
	ATHENA_IN_CI=false \
	ATHENA_GPG_ENABLED=$(ATHENA_GPG_ENABLED) \
	BIN_MODE=$(ATHENA_BIN_MODE) \
	ATHENA_E2E_TEST=false \
	ATHENA_APPLICATION_NAMESPACES=$(ATHENA_APPLICATION_NAMESPACES) \
		goreman -f $(ATHENA_PROCFILE) start ${ATHENA_START}


.PHONY: dep-ui
dep-ui: test-tools-image
	$(call run-in-test-client,make dep-ui-local)

dep-ui-local:
	cd ui && yarn install

.PHONY: cli
cli: test-tools-image
	$(call run-in-test-client, GOOS=${HOST_OS} GOARCH=${HOST_ARCH} make cli-local)

.PHONY: cli-local
cli-local: clean-debug
	CGO_ENABLED=${CGO_FLAG} GODEBUG="tarinsecurepath=0,zipinsecurepath=0" go build -gcflags="all=-N -l" $(COVERAGE_FLAG) -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/${CLI_NAME} ./cmd

# Cleans VSCode debug.test files from sub-dirs to prevent them from being included in by golang embed
.PHONY: clean-debug
clean-debug:
	-find ${CURRENT_DIR} -name debug.test -exec rm -f {} +

.PHONY: clean
clean: clean-debug
	-rm -rf ${CURRENT_DIR}/dist

.PHONY: lint-ui
lint-ui: test-tools-image
	$(call run-in-test-client,make lint-ui-local)

.PHONY: lint-ui-local
lint-ui-local:
	cd ui && yarn lint




# Build the UI
.PHONY: build-ui
build-ui:
	DOCKER_BUILDKIT=1 $(DOCKER) build -t athena-ui --platform=$(TARGET_ARCH) --target athena-ui .
	find ./ui/dist -type f -not -name gitkeep -delete
	$(DOCKER) run -v ${CURRENT_DIR}/ui/dist/app:/tmp/app --rm -t athena-ui sh -c 'cp -r ./dist/app/* /tmp/app/'

# Build the image
.PHONY: image
ifeq ($(DEV_IMAGE), true)
# The "dev" image builds the binaries from the users desktop environment (instead of in Docker)
# which speeds up builds. Dockerfile.dev needs to be copied into dist to perform the build, since
# the dist directory is under .dockerignore.
IMAGE_TAG="dev-$(shell git describe --always --dirty)"
image: build-ui
	DOCKER_BUILDKIT=1 $(DOCKER) build --platform=$(TARGET_ARCH) -t athena-base --target athena-base .
	CGO_ENABLED=${CGO_FLAG} GOOS=linux GOARCH=amd64 GODEBUG="tarinsecurepath=0,zipinsecurepath=0" go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/athena ./cmd
	ln -sfn ${DIST_DIR}/athena ${DIST_DIR}/athena-server
	cp Dockerfile.dev dist
	DOCKER_BUILDKIT=1 $(DOCKER) build --platform=$(TARGET_ARCH) -t $(IMAGE_PREFIX)athena:$(IMAGE_TAG) -f dist/Dockerfile.dev dist
else
image:
	DOCKER_BUILDKIT=1 $(DOCKER) build -t $(IMAGE_PREFIX)athena:$(IMAGE_TAG) --platform=$(TARGET_ARCH) .
endif
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then $(DOCKER) push $(IMAGE_PREFIX)athena:$(IMAGE_TAG) ; fi


.PHONY: athena-all
athena-all: clean-debug
	CGO_ENABLED=${CGO_FLAG} GOOS=${GOOS} GOARCH=${GOARCH} GODEBUG="tarinsecurepath=0,zipinsecurepath=0" go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/${BIN_NAME} ./cmd
