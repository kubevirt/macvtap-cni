#!/usr/bin/env bash

set -ex

source ./cluster/kubevirtci.sh

go test ./tests --kubeconfig $(kubevirtci::kubeconfig) -v --ginkgo.v
