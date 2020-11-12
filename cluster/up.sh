#!/usr/bin/env bash

set -ex

source ./cluster/cluster.sh
cluster::install

$(cluster::path)/cluster-up/up.sh

set +ex
echo '==============================================================================='
echo 'The cluster is ready!'
echo 'Use following command to install macvtap-cni on the cluster:'
echo '  make cluster-sync'
echo 'Use following command to access cluster API:'
echo '  ./cluster/kubectl.sh get nodes'
echo 'Use following command to ssh into cluster node:'
echo '  ./cluster/cli.sh ssh node01'
