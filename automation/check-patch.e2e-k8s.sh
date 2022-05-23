#!/usr/bin/env bash

# This script should be able to execute lifecycle functional tests against
# Kubernetes cluster on any environment with basic dependencies listed in
# check-patch.packages installed and docker running.
#
# yum -y install automation/check-patch.packages
# automation/check-patch.e2e-lifecycle-k8s.sh

set -xe

teardown() {
    make cluster-down
}

main() {
    export KUBEVIRT_PROVIDER='k8s-1.23'

    source automation/check-patch.setup.sh
    cd ${TMP_PROJECT_PATH}

    make cluster-down
    make cluster-up
    trap teardown EXIT SIGINT SIGTERM
    make cluster-sync
    make test/e2e
}

[[ "${BASH_SOURCE[0]}" == "$0" ]] && main "$@"
