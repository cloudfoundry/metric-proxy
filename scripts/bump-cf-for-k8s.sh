#!/usr/bin/env bash

set -ex

CF_FOR_K8s_DIR="${CF_FOR_K8s_DIR:-${HOME}/workspace/cf-for-k8s/}"
REPO_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && cd .. && pwd )"

pushd "${CF_FOR_K8s_DIR}"
  vendir sync -d config/_ytt_lib/github.com/cloudfoundry/metric-proxy="${REPO_DIR}"
popd

