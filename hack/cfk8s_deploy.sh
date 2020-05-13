#!/bin/bash

# run tests
go test ./... -race

# version it

# build image
base="ubuntu:bionic"
golang="https://dl.google.com/go/go1.13.8.linux-amd64.tar.gz"
repository="oratos"
tag="dev"

if [[ $VMWARE_BUILD == true ]]; then
  base="cnabu-docker-local.artifactory.eng.vmware.com/base/ubuntu:bionic-vmware-2019-11-01-21-34-33"
  golang="https://artifactory.eng.vmware.com:443/golang-dist/go1.13.8.linux-amd64.tar.gz"
  repository="epks-docker-local.artifactory.eng.vmware.com/oratos"
fi

image_name="${repository}/metric-proxy:${tag}"

docker build \
  --build-arg BASE_IMAGE="$base" \
  --build-arg GOLANG_SOURCE="$golang" \
  -t ${image_name} \
  .

# push image
docker push ${image_name}

# get env vars
#secret="$(kubectl get serviceaccount default -n cf-system -o json | jq -Mr '.secrets[].name | select(contains("token"))')"
#token="$(kubectl get secret ${secret} -n cf-system -o json | jq -Mr '.data.token' | base64 -D)"

# push to targeted environment
#cf push \
#  --docker-image ${image_name} \
#  --health-check-type process \
#  --var "API_SERVER=https://10.39.240.1:443" \
#  --var "TOKEN=${token}" \
#  --var "ADDR=:8080" \
#  --var "APP_SELECTOR=cloudfoundry.org/process_guid" \
#  --var "NAMESPACE=cf-workloads" \
#  --var "QUERY_TIMEOUT=5" \
#  metric-proxy
# API_SERVER=KUBERNETES_SERVICE_HOST=10.39.240.1

# configure CAPI to point to new app
