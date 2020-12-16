#!/usr/bin/env bash

set -ex

DOCKER_ORG=${DOCKER_ORG:-cloudfoundry}
REPO_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && cd .. && pwd )"
DEPLAB=${DEPLAB:-true}

function updateConfigValues {
    taggedImage=$1
    yttValuesRef=$2

    docker pull $DOCKER_ORG/$taggedImage
    imageRef="$(docker image inspect $DOCKER_ORG/$taggedImage --format '{{index .RepoDigests 0}}')"
    sed -i'' -e "s| $yttValuesRef:.*| $yttValuesRef: \"$imageRef\"|" ${REPO_DIR}/config/values.yml
}

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

    updateConfigValues $image:latest $yttValuesRef
}

