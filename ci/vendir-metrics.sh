#!/bin/bash
set -exc
pushd metric-proxy
  # Do we need this?
  new_metric_proxy_image_digest=$(cat ../metric-proxy-docker/digest)
  METRIC_PROXY_DIR="$(cd $(dirname $0) && pwd)"
popd
pushd cf-for-k8s
  vendir sync -d config/_ytt_lib/github.com/cloudfoundry/metric-proxy="${METRIC_PROXY_DIR}"
popd
