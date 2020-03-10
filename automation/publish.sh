#!/bin/bash -xe

# This script publishes macvtap-cni by default at quay.io/kubevirt
# organization.
# To publish elsewhere export the following env vars
# IMAGE_REGISTRY
# IMAGE_REPO
# IMAGE_TAG
# To run it just do proper docker login and automation/publish.sh

source automation/check-patch.setup.sh
cd ${TMP_PROJECT_PATH}

IMAGE_TAG=${IMAGE_TAG:-$(git log -1 --pretty=%h)-$(date +%s)}
make docker-build 
make docker-tag-latest
make docker-push
IMAGE_TAG=latest make docker-push
