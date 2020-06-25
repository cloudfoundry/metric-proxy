#!/usr/bin/env bash
set -ex

DOCKER_ORG=${DOCKER_ORG:-oratos}
REPO_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && cd .. && pwd )"
DEPLAB=${DEPLAB:-true}

function buildAndReplaceImage {
    image=$1
    srcFolder=$2
    dockerfile=$3
    yttValuesRef=$4

    docker build -t $DOCKER_ORG/$image:latest $srcFolder -f $srcFolder/$dockerfile
    if [ -n "$DEPLAB" ]; then
        rm -rf /tmp/$image
        deplab --image $DOCKER_ORG/$image:latest --git ${REPO_DIR} --output-tar /tmp/$image --tag $DOCKER_ORG/$image:latest
        docker load -i /tmp/$image
    fi
    docker push $DOCKER_ORG/$image:latest

    imageRef="$(docker image inspect $DOCKER_ORG/$image:latest --format '{{index .RepoDigests 0}}')"
    sed -i'' -e "s| $yttValuesRef:.*| $yttValuesRef: \"$imageRef\"|" ${REPO_DIR}/config/values.yml
}

buildAndReplaceImage metric-proxy ${REPO_DIR} Dockerfile metric_proxy
