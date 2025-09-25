#!/usr/bin/env bash

set -ex

source ./cluster/cluster.sh
cluster::install

export KUBEVIRT_WITH_CNAO=true
export KUBVIRT_WITH_CNAO_SKIP_CONFIG=true
$(cluster::path)/cluster-up/up.sh

echo 'Deploy CNAO CR'
./cluster/kubectl.sh create -f ./hack/cnao/cnao-cr.yaml
echo 'Wait for cluster operator'
./cluster/kubectl.sh wait networkaddonsconfig cluster --for condition=Available --timeout=800s

set +ex
echo '==============================================================================='
echo 'The cluster is ready!'
echo 'Use following command to install macvtap-cni on the cluster:'
echo '  make cluster-sync'
echo 'Use following command to access cluster API:'
echo '  ./cluster/kubectl.sh get nodes'
echo 'Use following command to ssh into cluster node:'
echo '  ./cluster/cli.sh ssh node01'
