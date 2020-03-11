CNI_MOUNT_PATH ?= /opt/cni/bin

IMAGE_REGISTRY ?= quay.io/kubevirt
IMAGE_NAME ?= macvtap-cni
IMAGE_TAG ?= latest

TARGETS = \
	goimports-format \
	goimports-check \
	whitespace-format \
	whitespace-check \
	vet

# Make does not offer a recursive wildcard function, so here's one:
rwildcard=$(wildcard $1$2) $(foreach d,$(wildcard $1*),$(call rwildcard,$d/,$2))

# Gather needed source files and directories to create target dependencies
directories=$(filter-out ./ ./vendor/ ./_out/ ./_kubevirtci/ ,$(sort $(dir $(wildcard ./*/))))
all_sources=$(call rwildcard,$(directories),*) $(filter-out $(TARGETS), $(wildcard *))
go_sources=$(call rwildcard,cmd/,*.go) $(call rwildcard,pkg/,*.go) $(call rwildcard,tests/,*.go)

# Configure Go
export GOOS=linux
export GOARCH=amd64
export CGO_ENABLED=0
export GO111MODULE=on
export GOFLAGS=-mod=vendor

.ONESHELL:

all: format check

check: goimports-check whitespace-check vet test/unit

format: goimports-format whitespace-format

goimports-check: $(go_sources)
	go run ./vendor/golang.org/x/tools/cmd/goimports -d ./pkg ./cmd ./tests
	touch $@

goimports-format: $(go_sources)
	go run ./vendor/golang.org/x/tools/cmd/goimports -w ./pkg ./cmd ./tests
	touch $@

whitespace-check: $(all_sources)
	go run ./vendor/golang.org/x/tools/cmd/goimports -d ./pkg ./cmd ./tests
	touch $@

whitespace-format: $(all_sources)
	go run ./vendor/golang.org/x/tools/cmd/goimports -w ./pkg ./cmd ./tests
	touch $@

vet: $(go_sources)
	go vet ./pkg/... ./cmd/... ./tests/...
	touch $@

docker-build:
	docker build -t ${IMAGE_REGISTRY}/${IMAGE_NAME}:${IMAGE_TAG} -f ./cmd/Dockerfile .

docker-push:
	docker push ${IMAGE_REGISTRY}/${IMAGE_NAME}:${IMAGE_TAG}

docker-tag-latest:
	docker tag ${IMAGE_REGISTRY}/${IMAGE_NAME}:latest ${IMAGE_REGISTRY}/${IMAGE_NAME}:${IMAGE_TAG}

cluster-up:
	./cluster/up.sh

cluster-down:
	./cluster/down.sh

cluster-sync:
	./cluster/sync.sh

test/e2e:
	./hack/functest.sh

test/unit:
	@if ! [ "$$(id -u)" = 0 ]; then
		@echo "You are not root, run this target as root please"
		exit 1
	fi
	go test ./cmd/... ./pkg/... -v --ginkgo.v

manifests:
	IMAGE_REGISTRY=$(IMAGE_REGISTRY) IMAGE_NAME=$(IMAGE_NAME) IMAGE_TAG=$(IMAGE_TAG) CNI_MOUNT_PATH=$(CNI_MOUNT_PATH) ./hack/generate-manifests.sh

vendor:
	go mod tidy
	go mod vendor

release: IMAGE_TAG = $(shell hack/version.sh)
release: docker-build docker-push

.PHONY: \
	all \
	check \
	cluster-up \
	cluster-down \
	cluster-sync \
	docker-build \
	docker-push \
	format \
	manifests \
	test/unit \
	vendor
