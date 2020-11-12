#!/usr/bin/env bash

source ./cluster/cluster.sh
cluster::install

$(cluster::path)/cluster-up/cli.sh "$@"
