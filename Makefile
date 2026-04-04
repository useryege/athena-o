
CURRENT_DIR := $(shell pwd)
DIST_DIR := $(CURRENT_DIR)/dist
HOST_OS := $(shell go env GOOS)
HOST_ARCH := $(shell go env GOARCH)
VOLUME_MOUNT := $(shell if test "$(go env GOOS)" = "darwin"; then echo ":delegated"; elif test selinuxenabled; then echo ":delegated"; else echo ""; fi)
GOPATH ?= $(shell if test -x `which go`; then go env GOPATH; else echo "$(HOME)/go"; fi)
GOCACHE ?= $(HOME)/.cache/go-build

DOCKER ?= docker
ifneq ($(DOCKER),docker)
$(error Only Docker is supported. Please run make with DOCKER=docker)
endif

DOCKER_SRCDIR ?= $(GOPATH)/src
DOCKER_WORKDIR ?= /go/src/github.com/useryege/athena
DOCKER_NETWORK ?= default
ifeq ($(DOCKER_NETWORK),default)
DOCKER_NETWORK_ARG =
else
DOCKER_NETWORK_ARG = --network $(DOCKER_NETWORK)
endif

TEST_TOOLS_NAMESPACE ?=
TEST_TOOLS_IMAGE = athena-test-tools
TEST_TOOLS_TAG ?= latest
ifdef TEST_TOOLS_NAMESPACE
TEST_TOOLS_PREFIX = $(TEST_TOOLS_NAMESPACE)/
endif

ifeq ("$(PWD)","$(GOPATH)/src/github.com/useryege/athena")
DOCKER_SRC_MOUNT = "$(DOCKER_SRCDIR):/go/src$(VOLUME_MOUNT)"
else
DOCKER_SRC_MOUNT = "$(PWD):/go/src/github.com/useryege/athena$(VOLUME_MOUNT)"
endif

CONTAINER_UID := $(shell id -u)
CONTAINER_GID := $(shell id -g)
SUDO ?=
ARGOCD_LINT_GOGC ?= 20

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
	@echo "ARGOCD_LINT_GOGC=$(ARGOCD_LINT_GOGC)"

define run-in-test-client
	$(SUDO) $(DOCKER) run --rm -it \
	  --name argocd-test-client \
		-u $(CONTAINER_UID):$(CONTAINER_GID) \
		-e HOME=/home/user \
		-e GOPATH=/go \
		-e ARGOCD_E2E_K3S=$(ARGOCD_E2E_K3S) \
		-e GITHUB_TOKEN \
		-e GOCACHE=/tmp/go-build-cache \
		-e ARGOCD_LINT_GOGC=$(ARGOCD_LINT_GOGC) \
		-v $(DOCKER_SRC_MOUNT) \
		-v $(GOPATH)/pkg/mod:/go/pkg/mod$(VOLUME_MOUNT) \
		-v $(GOCACHE):/tmp/go-build-cache$(VOLUME_MOUNT) \
		-v $(HOME)/.kube:/home/user/.kube$(VOLUME_MOUNT) \
		-w $(DOCKER_WORKDIR) \
		$(DOCKER_NETWORK_ARG) \
		$(TEST_TOOLS_PREFIX)$(TEST_TOOLS_IMAGE):$(TEST_TOOLS_TAG) \
		bash -c "$(1)"
endef

# Installs all tools required to build and test ArgoCD locally
.PHONY: install-tools-local
install-tools-local: install-test-tools-local install-codegen-tools-local install-go-tools-local

# Installs all tools required for running unit & end-to-end tests (Linux packages)
.PHONY: install-test-tools-local
install-test-tools-local:
	./hack/install.sh kustomize
	./hack/install.sh helm
	./hack/install.sh gotestsum
	./hack/install.sh oras

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

.PHONY: notification-catalog
notification-catalog:
	go run ./hack/gen-catalog catalog

.PHONY: notification-docs
notification-docs:
	go run ./hack/gen-docs
	go run ./hack/gen-catalog docs

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

# original codegen-local
# .PHONY: codegen-local
# codegen-local: mod-vendor-local mockgen gogen protogen clientgen clidocsgen  manifests-local notification-docs notification-catalog
# 	rm -rf vendor/

# new codegen-local
.PHONY: codegen-local
codegen-local: mod-vendor-local mockgen gogen protogen clientgen clidocsgen manifests-local
	rm -rf vendor/

.PHONY: codegen-local-fast
codegen-local-fast: mockgen gogen protogen-fast clientgen clidocsgen manifests-local notification-docs notification-catalog


.PHONY: test-tools-image
test-tools-image:
ifndef SKIP_TEST_TOOLS_IMAGE
	$(SUDO) $(DOCKER) build --build-arg UID=$(CONTAINER_UID) -t $(TEST_TOOLS_PREFIX)$(TEST_TOOLS_IMAGE) -f test/container/Dockerfile .
	$(SUDO) $(DOCKER) tag $(TEST_TOOLS_PREFIX)$(TEST_TOOLS_IMAGE) $(TEST_TOOLS_PREFIX)$(TEST_TOOLS_IMAGE):$(TEST_TOOLS_TAG)
endif

.PHONY: codegen
codegen: test-tools-image
	$(call run-in-test-client,make codegen-local)