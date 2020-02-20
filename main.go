package main

import (
	"log"
	"net"

	"code.cloudfoundry.org/log-cache/pkg/rpc/logcache_v1"
	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
	"k8s.io/metrics/pkg/client/clientset/versioned"

	"github.com/loggregator/metric-proxy/pkg/metrics"
)

func main() {
	cfg, err := LoadConfig()
	if err != nil {
		log.Fatalf("invalid configuration: %s", err)
	}

	// TODO: handle errors
	fetcher, _ := createMetricsFetcher(cfg)
	c := &metrics.Proxy{
		GetMetrics: fetcher,
		AddEmptyDiskEnvelope: true,
	}

	s := grpc.NewServer(grpc.Creds(cfg.TLS.Credentials("log-cache")))
	logcache_v1.RegisterEgressServer(s, c)

	lis, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	panic(s.Serve(lis))
}

func createMetricsFetcher(cfg *Config) (metrics.Fetcher, error) {
	restConfig := &rest.Config{
		Host:        cfg.APIServer,
		APIPath:     "/apis/metrics.k8s.io/v1beta1/pods",
		BearerToken: cfg.Token,
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: true,
		},
	}

	c, err := versioned.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	return func() (*v1beta1.PodMetricsList, error) {
		return c.MetricsV1beta1().PodMetricses(cfg.Namespace).List(v1.ListOptions{
			LabelSelector:  "app=" + cfg.AppSelector,
			TimeoutSeconds: &cfg.QueryTimeout,
		})
	}, nil
}
