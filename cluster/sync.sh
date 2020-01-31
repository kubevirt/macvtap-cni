#!/usr/bin/env bash

set -ex

source ./cluster/kubevirtci.sh
kubevirtci::install

registry_port=$(./cluster/cli.sh ports registry | tr -d '\r')
registry=localhost:$registry_port

IMAGE_REGISTRY=$registry make docker-build
IMAGE_REGISTRY=$registry make docker-push

destination=_out/manifests
rm -rf $destination
mkdir -p $destination
DESTINATION=$destination IMAGE_REGISTRY=registry:5000 make manifests

# TODO: Remove old components and apply new ones
