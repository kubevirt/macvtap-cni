#!/usr/bin/env bash

set -ex

source ./cluster/kubevirtci.sh

KUBECONFIG=${KUBECONFIG:-$(kubevirtci::kubeconfig)}
go test ./tests/e2e --kubeconfig ${KUBECONFIG} -ginkgo.v
