#!/bin/bash
set -Eeuo pipefail; [ -n "${DEBUG:-}" ] && set -x

function set_pipeline {
    echo "setting pipeline for metric-proxy"
    SCRIPT_DIR="$(cd $(dirname $0) && pwd -P)"
    metric_proxy_creds_file=$(mktemp)
    vault read -field=vars_file /secret/concourse/main/metric-proxy-pipeline > $metric_proxy_creds_file
    fly -t loggregator set-pipeline -p metric-proxy \
        -l "${metric_proxy_creds_file}" \
        -c "${SCRIPT_DIR}/validation.yml"
}

function sync_fly {
    fly -t loggregator sync
}

function main {
    sync_fly
    set_pipeline
}
main
