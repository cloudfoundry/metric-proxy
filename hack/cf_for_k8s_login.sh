#!/bin/bash

zone_name=${1:-sloans-lake}
cf_domain="${zone_name}.loggr.cf-app.com"

cf_admin_password="$(vault read --field=cf-values.yml secret/envs/cf4k8s/${zone_name} | yq r - cf_admin_password)"

echo "Targeting cf on ${zone_name}"

cf api --skip-ssl-validation https://api.${cf_domain}
cf auth admin ${cf_admin_password}
cf create-org test-org
cf create-space -o test-org test-space
cf target -o test-org -s test-space
