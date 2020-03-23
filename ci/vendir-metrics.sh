#!/bin/bash
set -ex

METRIC_PROXY_DIR="$(cd $(dirname $0) && cd .. && pwd)"

pushd cf-for-k8s
  vendir sync -d config/_ytt_lib/github.com/cloudfoundry/metric-proxy="${METRIC_PROXY_DIR}"
popd
