#!/bin/bash

zone_name=${1:-sloans-lake}
gcp_key=${2:-/tmp/sa.json}
cf_domain="${zone_name}.loggr.cf-app.com"

echo "Creating a GCP cluster with the domain name ${cf_domain}"

gcloud auth login
gcloud container clusters create ${zone_name} --region us-central1 \
    --no-enable-cloud-logging --no-enable-cloud-monitoring \
    --machine-type=n1-standard-4 --enable-network-policy --cluster-version=1.15.11
gcloud container clusters get-credentials ${zone_name} --region us-central1

echo "Deploying cf-for-k8s on the new cluster"

pushd ~/workspace/cf-for-k8s
  echo "Deploying with cf-for-k8s branch $(git rev-parse --abbrev-ref HEAD)"
  ./hack/generate-values.sh -g ${gcp_key} -d ${cf_domain} > /tmp/cf-values.yml
  ytt -f config -f /tmp/cf-values.yml > /tmp/cf-for-k8s-rendered.yml
  kapp deploy -a cf -f /tmp/cf-for-k8s-rendered.yml -y
  ./hack/update-gcp-dns.sh ${cf_domain} ${zone_name}
popd

echo "Targeting cf"

cf api --skip-ssl-validation https://api.${cf_domain}
cf auth admin $(head /tmp/cf-values.yml | grep cf_admin_password | awk '{print $2}')
cf create-org test-org
cf create-space -o test-org test-space
cf target -o test-org -s test-space

cat <<EOF
Done!

Your cf-for-k8s cluster is up! You're targeting test-org and test-space.

When you're done with it, run
    \`gcloud container clusters delete ${zone_name} --region us-central1\`
EOF
