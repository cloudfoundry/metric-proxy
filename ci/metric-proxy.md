# cf-k8s-metric-proxy

## Purpose
This pipeline serves two purposes: testing changes and cutting releases/images
in the [cf-k8s-metric-proxy CI pipeline](https://release-integration.ci.cf-app.com/teams/main/pipelines/cf-k8s-metric-proxy).

## metric-proxy-unit-tests
Runs all the Golang unit tests in the `metric-proxy` source.

## test-metric-proxy-bump-on-cf-for-k8s job
This is an integration test to ensure that metric-proxy changes work on top of cf-for-k8s main branch. This is done via
the [smoke-tests](https://github.com/cloudfoundry/cf-for-k8s/blob/develop/tests/smoke/smoke_test.go).

## metric-proxy-cut-{major, minor, patch}
These jobs bump the appropriate number in [metric-proxy/version](https://github.com/cloudfoundry/metric-proxy/blob/main/version) file that gets included as [Concourse semver resource](https://github.com/concourse/semver-resource). This is used in the pipeline to cut a new release. Depending on version cut, the appropriate job needs to be manually triggered to bump the version in the file. This needs to be followed up with the next job to `create-metric-proxy-release` at this new version.

## create-metric-proxy-release
This job builds and [deplabs](https://github.com/vmware-tanzu/dependency-labeler) the metric-proxy image as well as cuts a Github
release. The job is also responsible for writing the version of metric-proxy to the `config/values/_defaults.yml` and `config/values/image.yml` files.

