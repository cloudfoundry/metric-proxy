# Releasing Metric Proxy

Note that metric-proxy does NOT use a develop->main workflow. Commits are pushed and PRs are merged directly to main, letting the pipeline test changes after landing on main.

## Steps to release

1. Make sure metric-proxy `main` has what you need.
1. Head to the [cf-k8s-metric-proxy-validation](https://release-integration.ci.cf-app.com/teams/main/pipelines/cf-k8s-metric-proxy-validation) pipeline and make sure the `test-metric-proxy-bump-on-cf-for-k8s` job is green
1. Bump the version appropriately using the `metric-proxy-cut-{major|minor|patch}` job.
1. Finish by triggering the create-release job
    - Takes care of OSL and runs `deplab`
    - Updates the `values.yml` file to point to new image sha created in the job
1. Head to [metric-proxy releases](https://github.com/cloudfoundry/metric-proxy/releases) and edit the new draft release. Add relevant release notes and then publish the release.
    - Note: the `values.yml` Asset on the release contains the updated metric-proxy image.

### Pipeline resources
* The pipeline is flown to https://release-integration.ci.cf-app.com/teams/main/pipelines/cf-k8s-metric-proxy-validation via `cd ~/workspace/metric-proxy && ./ci/configure`
* Yaml config lives in https://github.com/cloudfoundry/metric-proxy/blob/main/ci/validation.yml

### Building images

#### statsd_exporter
This image is built in Concourse; however, we do not automate bumping this image in wherever it is used in `cf-for-k8s`.
As a result, you will have to check to see if this job has ran since you last updated the `statsd_exporter` image reference in `cf-for-k8s`: https://release-integration.ci.cf-app.com/teams/main/pipelines/build-component-images/jobs/build-statsd_exporter-cf-for-k8s-image/

1. Grab the image reference output from the latest build: https://release-integration.ci.cf-app.com/teams/main/pipelines/build-component-images/jobs/build-statsd_exporter-cf-for-k8s-image/
1. Update the reference in the following file with that new image reference: https://github.com/cloudfoundry/cf-for-k8s/blob/develop/config/values/images.yml
    - To validate that a new image is indeed ready to be bumped, the image reference from the latest Concourse build should **not** match the reference in this file
1. Commit those changes and make a PR back to `cf-for-k8s` for it to be merged in

