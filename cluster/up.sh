#!/usr/bin/env bash

set -ex

source ./cluster/kubevirtci.sh
kubevirtci::install

$(kubevirtci::path)/cluster-up/up.sh

set +ex
echo '==============================================================================='
echo 'The cluster is ready!'
echo 'Use following command to install macvtap-cni on the cluster:'
echo '  make cluster-sync'
echo 'Use following command to access cluster API:'
echo '  ./kubevirtci/cluster-up/kubectl.sh get nodes'
echo 'Use following command to ssh into cluster node:'
echo '  ./kubevirtci/cluster-up/cli.sh ssh node01'
