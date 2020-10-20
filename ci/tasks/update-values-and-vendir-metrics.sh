#!/bin/bash
set -ex

# this is the CI version of updateConfigValues
imageRef="$(cat metric-proxy-docker/repository)@$(cat metric-proxy-docker/digest)"
sed -i'' -e "s| metric_proxy:.*| metric_proxy: \"$imageRef\"|" metric-proxy/config/values.yml

# this is the CI version of hack/bump-cf-for-k8s.sh
pushd cf-for-k8s
  vendir sync -d config/metrics/_ytt_lib/metric-proxy=metric-proxy/config
popd
