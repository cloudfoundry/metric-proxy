#!/bin/bash
REPO_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && cd .. && pwd )"

pushd $REPO_DIR
    make test

    repository="oratos"
    tag="dev"
    image_name="${repository}/metric-proxy:${tag}"

    docker build -t ${image_name} .
    docker push ${image_name}

    source ${REPO_DIR}/hack/helpers.sh

    updateConfigValues metric-proxy:${tag} metric_proxy
popd

