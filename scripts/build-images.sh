#!/usr/bin/env bash

source $REPO_DIR/helpers.sh

buildAndReplaceImage metric-proxy ${REPO_DIR} Dockerfile metric_proxy
