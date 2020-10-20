#!/usr/bin/env bash

set -ex

# this is the local version of ci/tasks/vendir-metrics.sh
CF_FOR_K8s_DIR="${CF_FOR_K8s_DIR:-${HOME}/workspace/cf-for-k8s/}"
REPO_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && cd .. && pwd )"

pushd "${CF_FOR_K8s_DIR}"
  vendir sync -d config/metrics/_ytt_lib/metric-proxy="${REPO_DIR}/config"
popd
