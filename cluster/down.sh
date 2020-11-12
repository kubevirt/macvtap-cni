#!/usr/bin/env bash

set -ex

source ./cluster/cluster.sh
cluster::install

$(cluster::path)/cluster-up/down.sh
