#!/usr/bin/env bash

export KUBEVIRT_PROVIDER='k8s-1.14.6'

KUBEVIRTCI_VERSION='0e5b027098796137a9b95aed57943061e185bfcd'
KUBEVIRTCI_PATH="${PWD}/_kubevirtci"

function kubevirtci::install() {
    if [ ! -d ${KUBEVIRTCI_PATH} ]; then
        git clone https://github.com/kubevirt/kubevirtci.git ${KUBEVIRTCI_PATH}
        (
            cd ${KUBEVIRTCI_PATH}
            git checkout ${KUBEVIRTCI_VERSION}
        )
    fi
}

function kubevirtci::path() {
    echo -n ${KUBEVIRTCI_PATH}
}

function kubevirtci::kubeconfig() {
    echo -n ${KUBEVIRTCI_PATH}/_ci-configs/${KUBEVIRT_PROVIDER}/.kubeconfig
}
