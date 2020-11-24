#!/usr/bin/env bash

set -ex

source ./cluster/cluster.sh
cluster::install

registry_port=$(./cluster/cli.sh ports registry | tr -d '\r')
registry=localhost:$registry_port

IMAGE_REGISTRY=$registry make docker-build
IMAGE_REGISTRY=$registry make docker-push

destination=_out/manifests
rm -rf $destination
mkdir -p $destination
DESTINATION=$destination IMAGE_REGISTRY=registry:5000 make manifests

./cluster/kubectl.sh delete --ignore-not-found configmap macvtap-deviceplugin-config
./cluster/kubectl.sh delete --ignore-not-found ds macvtap-cni

./cluster/kubectl.sh create -f examples/macvtap-deviceplugin-config-default.yaml
./cluster/kubectl.sh create -f _out/manifests/macvtap.yaml
