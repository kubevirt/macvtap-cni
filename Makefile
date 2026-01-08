CNI_MOUNT_PATH ?= /opt/cni/bin
DEVICE_PLUGIN_CONFIG_MAP_NAME ?= macvtap-deviceplugin-config

IMAGE_NAME ?= macvtap-cni
IMAGE_REGISTRY ?= quay.io/kubevirt
IMAGE_PULL_POLICY ?= Always
IMAGE_TAG ?= latest

NAMESPACE ?= default

TARGETS = \
	goimports-format \
	goimports-check \
	whitespace-format \
	whitespace-check \
	vet

OCI_BIN ?= docker

# tools
GITHUB_RELEASE ?= $(GOBIN)/github-release
PLATFORM_LIST ?= linux/amd64,linux/s390x,linux/arm64
ARCH := $(shell uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')
PLATFORMS ?= linux/${ARCH}
PLATFORMS := $(if $(filter all,$(PLATFORMS)),$(PLATFORM_LIST),$(PLATFORMS))
# Set the platforms for building a multi-platform supported image.
# Example:
# PLATFORMS ?= linux/amd64,linux/arm64,linux/s390x
# Alternatively, you can export the PLATFORMS variable like this:
# export PLATFORMS=linux/arm64,linux/s390x,linux/amd64
# or export PLATFORMS=all to automatically include all supported platforms.
DOCKER_BUILDER ?= macvtap-docker-builder

# Make does not offer a recursive wildcard function, so here's one:
rwildcard=$(wildcard $1$2) $(foreach d,$(wildcard $1*),$(call rwildcard,$d/,$2))

# Gather needed source files and directories to create target dependencies
directories=$(filter-out ./ ./vendor/ ./_out/ ./_kubevirtci/ ,$(sort $(dir $(wildcard ./*/))))
all_sources=$(call rwildcard,$(directories),*) $(filter-out $(TARGETS), $(wildcard *))
go_sources=$(call rwildcard,cmd/,*.go) $(call rwildcard,pkg/,*.go) $(call rwildcard,tests/,*.go)

# Configure Go
export GOOS=linux
export GOARCH=$(shell uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')
export CGO_ENABLED=0
export GO111MODULE=on
export GOFLAGS=-mod=vendor

BIN_DIR = $(CURDIR)/build/_output/bin/
export GOROOT=$(BIN_DIR)/go/
export GOBIN = $(GOROOT)/bin/
export PATH := $(GOBIN):$(PATH)

GO := $(GOBIN)/go

$(GO):
	hack/install-go.sh $(BIN_DIR)

.ONESHELL:

all: format check

check: goimports-check whitespace-check vet test/unit

format: goimports-format whitespace-format

goimports-check: $(go_sources) $(GO)
	$(GO) run ./vendor/golang.org/x/tools/cmd/goimports -d ./pkg ./cmd ./tests

goimports-format: $(go_sources) $(GO)
	$(GO) run ./vendor/golang.org/x/tools/cmd/goimports -w ./pkg ./cmd ./tests

whitespace-check: $(all_sources) $(GO)
	$(GO) run ./vendor/golang.org/x/tools/cmd/goimports -d ./pkg ./cmd ./tests

whitespace-format: $(all_sources)
	$(GO) run ./vendor/golang.org/x/tools/cmd/goimports -w ./pkg ./cmd ./tests

vet: $(go_sources) $(GO)
	$(GO) vet ./pkg/... ./cmd/... ./tests/...

docker-build:
ifeq ($(OCI_BIN),podman)
	$(MAKE) build-multiarch-macvtap-podman MACVTAP_IMAGE_TAGGED=$(IMAGE_REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)
else ifeq ($(OCI_BIN),docker)
	$(MAKE) build-multiarch-macvtap-docker MACVTAP_IMAGE_TAGGED=$(IMAGE_REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)
else
	$(error Unsupported OCI_BIN value: $(OCI_BIN))
endif

docker-push:
ifeq ($(OCI_BIN),podman)
	podman manifest push --tls-verify=false $(IMAGE_REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG) $(IMAGE_REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)
endif

docker-tag-latest:
	$(OCI_BIN) tag ${IMAGE_REGISTRY}/${IMAGE_NAME}:latest $(IMAGE_REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)

cluster-up:
	./cluster/up.sh

cluster-down:
	./cluster/down.sh

cluster-sync:
	./cluster/sync.sh

test/e2e: $(GO)
	GO=$(GO) ./hack/functest.sh

test/unit: $(GO)
	@if ! [ "$$(id -u)" = 0 ]; then
		@echo "You are not root, run this target as root please"
		exit 1
	fi
	# mount dynamic /dev to see new /dev/tapXX files (containers use static tmpfs /dev)
	[ "$$(findmnt -n -o FSTYPE /dev)" != devtmpfs ] && mount -t devtmpfs none /dev
	$(GO) test ./cmd/... ./pkg/... -v --ginkgo.v

manifests:
	IMAGE_REGISTRY=$(IMAGE_REGISTRY) IMAGE_NAME=$(IMAGE_NAME) IMAGE_TAG=$(IMAGE_TAG) CNI_MOUNT_PATH=$(CNI_MOUNT_PATH) NAMESPACE=$(NAMESPACE) IMAGE_PULL_POLICY=$(IMAGE_PULL_POLICY) DEVICE_PLUGIN_CONFIG_MAP_NAME=$(DEVICE_PLUGIN_CONFIG_MAP_NAME) ./hack/generate-manifests.sh

vendor: $(GO)
	$(GO) mod tidy
	$(GO) mod vendor

prepare-patch:
	./hack/prepare-release.sh patch
prepare-minor:
	./hack/prepare-release.sh minor
prepare-major:
	./hack/prepare-release.sh major

$(GITHUB_RELEASE): go.mod $(GO)
	$(GO) install ./vendor/github.com/aktau/github-release

release: IMAGE_TAG = $(shell hack/version.sh)
release: docker-build docker-push
release: $(GITHUB_RELEASE)
	TAG=$(IMAGE_TAG) GITHUB_RELEASE=$(GITHUB_RELEASE) DESCRIPTION=./version/description ./hack/release.sh

build-multiarch-macvtap-docker:
	PLATFORMS=$(PLATFORMS) MACVTAP_IMAGE_TAGGED=$(MACVTAP_IMAGE_TAGGED) DOCKER_BUILDER=$(DOCKER_BUILDER) ./hack/build-macvtap-docker.sh

build-multiarch-macvtap-podman:
	PLATFORMS=$(PLATFORMS) MACVTAP_IMAGE_TAGGED=$(MACVTAP_IMAGE_TAGGED) ./hack/build-macvtap-podman.sh

.PHONY: \
	all \
	build-multiarch-macvtap-docker \
	build-multiarch-macvtap-podman \
	check \
	cluster-up \
	cluster-down \
	cluster-sync \
	docker-build \
	docker-push \
	format \
	manifests \
	test/unit \
	vendor \
	vet \
	goimports-check \
	goimports-format \
	whitespace-check \
	whitespace-format
