#!/usr/bin/env bash
REPO_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && cd .. && pwd )"

source $REPO_DIR/scripts/helpers.sh

buildAndReplaceImage metric-proxy ${REPO_DIR} Dockerfile metric_proxy
