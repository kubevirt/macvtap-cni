IMAGE_REGISTRY ?= quay.io/kubevirt
IMAGE_NAME ?= macvtap-cni
IMAGE_TAG ?= latest

TARGETS = \
	goimports-format \
	goimports-check \
	whitespace-format \
	whitespace-check

# Make does not offer a recursive wildcard function, so here's one:
rwildcard=$(wildcard $1$2) $(foreach d,$(wildcard $1*),$(call rwildcard,$d/,$2))

# Gather needed source files and directories to create target dependencies
directories := $(filter-out ./ ./vendor/ ,$(sort $(dir $(wildcard ./*/))))
all_sources=$(call rwildcard,$(directories),*) $(filter-out $(TARGETS), $(wildcard *))

all: format check

check: goimports-check whitespace-check test/unit

format: goimports-format whitespace-format

goimports-check: $(all_sources)
	go run ./vendor/golang.org/x/tools/cmd/goimports -d ./pkg ./cmd ./test
	touch $@

goimports-format: $(all_sources)
	go run ./vendor/golang.org/x/tools/cmd/goimports -w ./pkg ./cmd ./test
	touch $@

whitespace-check: $(all_sources)
	go run ./vendor/golang.org/x/tools/cmd/goimports -d ./pkg ./cmd ./test
	touch $@

whitespace-format: $(all_sources)
	go run ./vendor/golang.org/x/tools/cmd/goimports -w ./pkg ./cmd ./test
	touch $@

docker-build:
	docker build -t ${IMAGE_REGISTRY}/${IMAGE_NAME}:${IMAGE_TAG} ./cmd

docker-push:
	docker push ${IMAGE_REGISTRY}/${IMAGE_NAME}:${IMAGE_TAG}

cluster-up:
	./cluster/up.sh

cluster-down:
	./cluster/down.sh

cluster-sync:
	./cluster/sync.sh

test/e2e:
	./hack/functest.sh

test/unit:
	go test ./cmd/... ./pkg/... -v --ginkgo.v

manifests:
	IMAGE_REGISTRY=$(IMAGE_REGISTRY) IMAGE_NAME=$(IMAGE_NAME) IMAGE_TAG=$(IMAGE_TAG) ./hack/generate-manifests.sh

vendor:
	go mod tidy
	go mod vendor

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
