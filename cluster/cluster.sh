#!/usr/bin/env bash


export KUBEVIRT_PROVIDER="${KUBEVIRT_PROVIDER:-k8s-1.19}"
export KUBEVIRTCI_TAG="2011240648-fb5563f"
# The CLUSTER_PATH var is used in cluster folder and points to the _kubevirtci where the cluster is deployed from.
CLUSTER_PATH=${CLUSTER_PATH:-"${PWD}/_kubevirtci/"}

function cluster::install() {
    if [ ! -d ${CLUSTER_PATH} ]; then
        git clone https://github.com/kubevirt/kubevirtci.git ${CLUSTER_PATH}
        (
            cd ${CLUSTER_PATH}
            git checkout ${KUBEVIRTCI_TAG}
        )
    fi
}

function cluster::path() {
    echo -n ${CLUSTER_PATH}
}

function cluster::kubeconfig() {
    echo -n ${CLUSTER_PATH}/_ci-configs/${KUBEVIRT_PROVIDER}/.kubeconfig
}
