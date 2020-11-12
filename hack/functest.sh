#!/usr/bin/env bash

set -ex

source ./cluster/cluster.sh

KUBECONFIG=${KUBECONFIG:-$(cluster::kubeconfig)}
${GO} test ./tests/e2e --kubeconfig ${KUBECONFIG} -ginkgo.v
