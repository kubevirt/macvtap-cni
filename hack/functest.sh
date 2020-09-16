#!/usr/bin/env bash

set -ex

source ./cluster/kubevirtci.sh

go test ./tests/e2e --kubeconfig $(kubevirtci::kubeconfig) -ginkgo.v
