module code.cloudfoundry.org/metric-proxy

go 1.13

require (
	code.cloudfoundry.org/go-envstruct v1.5.0
	code.cloudfoundry.org/go-loggregator v7.4.0+incompatible
	code.cloudfoundry.org/go-metric-registry v0.0.0-20200413202920-40d97c8804ec
	code.cloudfoundry.org/log-cache v2.3.1+incompatible
	code.cloudfoundry.org/rfc5424 v0.0.0-20180905210152-236a6d29298a // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.12.2 // indirect
	github.com/maxbrunsfeld/counterfeiter/v6 v6.3.0
	github.com/onsi/gomega v1.10.3
	github.com/prometheus/client_golang v1.5.1 // indirect
	golang.org/x/net v0.0.0-20201026091529-146b70c837a4
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d // indirect
	golang.org/x/time v0.0.0-20191024005414-555d28b269f0 // indirect
	google.golang.org/genproto v0.0.0-20200128133413-58ce757ed39b // indirect
	google.golang.org/grpc v1.27.0
	k8s.io/api v0.17.2
	k8s.io/apimachinery v0.17.2
	k8s.io/client-go v0.17.2
	k8s.io/metrics v0.17.2
	k8s.io/utils v0.0.0-20200122174043-1e243dd1a584 // indirect
)
