#!/usr/bin/env bash

set -e

zone_name=${1:-sloans-lake}
gcp_key=${2:-/tmp/sa.json}
cf_domain="${zone_name}.loggr.cf-app.com"

# these should be flags passed in, but would need to configure playing nice
# with positional args
LPASS_GCP_KEY="${LPASS_GCP_KEY:-Shared-Loggregator (Pivotal Only)/GCP Service Account Key}"
SKIP_CREATE_CLUSTER=${SKIP_CREATE_CLUSTER:-false}

function usage() {
  cat << HEREDOC

  Usage: $0 [ZONE_NAME GCP_KEY]

    If not provided, ZONE_NAME and GCP_KEY will default to sloans-lake and
    /tmp/sa.json, respectively.

    If the GCP_KEY does not exist, it will be fetched from Lastpass using
    the environment variable LPASS_GCP_KEY.

    To avoid creating a new cluster, set the SKIP_CREATE_CLUSTER environment
    variable. Example:

      SKIP_CREATE_CLUSTER=true $0

  optional arguments:
    -h, --help           show this help message and exit

HEREDOC
}

# process args
for arg in "$@"; do
  case "$arg" in
    -h|--help) usage; exit 0;;
    *) ;;
  esac
done

# verify args
if [[ ! -f ${gcp_key} ]]; then
  echo "GCP key ${gcp_key} not found. Loading from Lastpass into /tmp/sa.json"
  gcp_key="/tmp/sa.json"
  lpass show "$LPASS_GCP_KEY" --notes > $gcp_key
fi

if [[ "$SKIP_CREATE_CLUSTER" != true ]]; then
  echo "Creating a GCP cluster with the domain name ${cf_domain}"
  gcloud auth login
  gcloud container clusters create ${zone_name} --region us-central1 \
    --no-enable-cloud-logging --no-enable-cloud-monitoring \
    --machine-type=n1-standard-4 --enable-network-policy --cluster-version=1.15.11
fi

echo "Targeting GCP cluster with the domain name ${zone_name}"
gcloud container clusters get-credentials ${zone_name} --region us-central1

echo "Deploying to cf-for-k8s on the ${zone_name} cluster"
pushd ~/workspace/cf-for-k8s
  ./hack/generate-values.sh -g ${gcp_key} -d ${cf_domain} > /tmp/cf-values.yml
  vault write secret/envs/cf4k8s/${zone_name} cf-values.yml=@/tmp/cf-values.yml

  ytt -f config -f /tmp/cf-values.yml > /tmp/cf-for-k8s-rendered.yml
  kapp deploy -a cf -f /tmp/cf-for-k8s-rendered.yml -y || { echo "Failed to deploy cf-for-k8s. Exiting."; exit 1; }

  echo "Updating DNS records. Ensure that the ${zone_name} DNS record exists in GCP"
  ./hack/update-gcp-dns.sh ${cf_domain} ${zone_name} \
    || echo "Warning: failed to update GCP DNS. Attempting to continue, but targeting may fail."
popd

./hack/cf_for_k8s_login.sh ${zone_name} \
    || { echo "Failed to target cf-for-k8s at ${zone_name}." ; exit 2; }

cat <<EOF
Done!

Your cf-for-k8s cluster is up! You're targeting test-org and test-space.

When you're done with it, run
    \`gcloud container clusters delete ${zone_name} --region us-central1\`
EOF
