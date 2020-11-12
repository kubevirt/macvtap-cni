#!/usr/bin/env bash


export KUBEVIRT_PROVIDER="${KUBEVIRT_PROVIDER:-k8s-multus-1.13.3}"

KUBEVIRTCI_VERSION='9d224d0c22e9ed2ca7588ccf3a258d82e160b195'
# The CLUSTER_PATH var is used in cluster folder and points to the _kubevirtci where the cluster is deployed from.
CLUSTER_PATH=${CLUSTER_PATH:-"${PWD}/_kubevirtci/"}

function cluster::install() {
    if [ ! -d ${CLUSTER_PATH} ]; then
        git clone https://github.com/kubevirt/kubevirtci.git ${CLUSTER_PATH}
        (
            cd ${CLUSTER_PATH}
            git checkout ${KUBEVIRTCI_VERSION}
        )
    fi
}

function cluster::path() {
    echo -n ${CLUSTER_PATH}
}

function cluster::kubeconfig() {
    echo -n ${CLUSTER_PATH}/_ci-configs/${KUBEVIRT_PROVIDER}/.kubeconfig
}
