#!/usr/bin/env bash


export KUBEVIRT_PROVIDER="${KUBEVIRT_PROVIDER:-k8s-multus-1.13.3}"

KUBEVIRTCI_VERSION='9d224d0c22e9ed2ca7588ccf3a258d82e160b195'
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
