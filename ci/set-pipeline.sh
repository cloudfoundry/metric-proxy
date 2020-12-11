#!/bin/bash
set -Eeuo pipefail; [ -n "${DEBUG:-}" ] && set -x

function set_pipeline {
    echo "setting pipeline for metric-proxy"
    SCRIPT_DIR="$(cd $(dirname $0) && pwd -P)"
    fly -t loggregator set-pipeline -p metric-proxy \
        -l <(lpass show 'Shared-CF-Oratos (Pivotal ONLY)/metric-proxy-pipeline' --notes) \
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
