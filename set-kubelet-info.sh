#!/bin/bash
set -ex
export SECRET="$(kubectl get serviceaccount default -n cf-system -o json | jq -Mr '.secrets[].name | select(contains("token"))')"
export TOKEN="$(kubectl get secret ${SECRET} -n cf-system -o json | jq -Mr '.data.token' | base64 -D)"
kubectl get secret ${SECRET} -n cf-system -o json | jq -Mr '.data["ca.crt"]' | base64 -D > /tmp/ca.crt

server=$(kubectl cluster-info | awk '/master/ {print $6}')
export API_SERVER="${server}:8443"

echo "API_SERVER=$API_SERVER"
echo "TOKEN=$TOKEN"
