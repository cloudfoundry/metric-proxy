#!/bin/bash
set -euo pipefail

trap "pkill dockerd" EXIT

start-docker &
echo 'until docker info; do sleep 5; done' >/usr/local/bin/wait_for_docker
chmod +x /usr/local/bin/wait_for_docker
timeout 300 wait_for_docker

<<<"$DOCKERHUB_PASSWORD" docker login --username "$DOCKERHUB_USERNAME" --password-stdin

metric-proxy/build/build.sh

image_ref="$(yq -r '.overrides[] | select(.image | test("/metric-proxy@")).newImage' metric-proxy/build/kbld.lock.yml)"
sed -i'' -e "s| metric_proxy:.*| metric_proxy: \"$image_ref\"|" metric-proxy/config/values/images.yml

pushd metric-proxy > /dev/null
  git config user.name "${GIT_COMMIT_USERNAME}"
  git config user.email "${GIT_COMMIT_EMAIL}"
  git add config/values/images.yml

  # dont make a commit if there aren't new images
  if ! git diff --cached --exit-code; then
    echo "committing!"
    git commit -m "images.yml updated by CI"
  else
    echo "no changes to images, not bothering with a commit"
  fi
popd > /dev/null

cp -R metric-proxy/. updated-metric-proxy/
