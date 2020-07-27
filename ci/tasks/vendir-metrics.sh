#!/bin/bash
set -ex

METRIC_PROXY_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && cd ../.. && pwd )"

pushd cf-for-k8s
  vendir sync -d config/metrics/_ytt_lib/metric-proxy="${METRIC_PROXY_DIR}"
popd
