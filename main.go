package main

import (
	"fmt"
	"log"
	"net"

	"code.cloudfoundry.org/go-envstruct"
	"code.cloudfoundry.org/log-cache/pkg/rpc/logcache_v1"
	"google.golang.org/grpc"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
	"k8s.io/metrics/pkg/client/clientset/versioned"

	"code.cloudfoundry.org/metric-proxy/pkg/metrics"
)

func main() {
	cfg, err := LoadConfig()
	if err != nil {
		log.Fatalf("invalid configuration: %s", err)
	}

	log.Println("starting metric-proxy...")
	defer log.Println("exiting metric-proxy...")
	envstruct.WriteReport(cfg)

	fetcher, err := createMetricsFetcher(cfg)
	if err != nil {
		log.Fatalf("cannot initialize metric fetcher: %v", err)
	}

	c := &metrics.Proxy{
		GetMetrics:           fetcher,
		AddEmptyDiskEnvelope: true,
	}

	s := grpc.NewServer(grpc.Creds(cfg.TLS.Credentials("metric-proxy")))
	logcache_v1.RegisterEgressServer(s, c)

	lis, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	panic(s.Serve(lis))
}

func createMetricsFetcher(cfg *Config) (metrics.Fetcher, error) {
	restConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	c, err := versioned.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	return func(appGuid string) (*v1beta1.PodMetricsList, error) {
		return c.MetricsV1beta1().PodMetricses(cfg.Namespace).List(v1.ListOptions{
			LabelSelector:  fmt.Sprintf("%s=%s", cfg.AppSelector, appGuid),
			TimeoutSeconds: &cfg.QueryTimeout,
		})
	}, nil
}
