package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"code.cloudfoundry.org/go-envstruct"
	"code.cloudfoundry.org/log-cache/pkg/rpc/logcache_v1"
	"code.cloudfoundry.org/metric-proxy/pkg/metrics"

	metricRegistry "code.cloudfoundry.org/go-metric-registry"
	"google.golang.org/grpc"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
	"k8s.io/metrics/pkg/client/clientset/versioned"
)

var (
	version = "dev-build"
	requestDurations metricRegistry.Histogram
)


func main() {
	loggr := log.New(os.Stderr, "", log.LstdFlags)
	loggr.Printf("starting metric-proxy %s...\n", version)
	defer loggr.Println("exiting metric-proxy...")

	cfg, err := LoadConfig()
	if err != nil {
		loggr.Fatalf("invalid configuration: %s", err)
	}

	err = envstruct.WriteReport(cfg)
	if err != nil {
		loggr.Fatalf("cannot report envstruct config: %v", err)
	}

	fetcher, err := createMetricsFetcher(cfg)
	if err != nil {
		loggr.Fatalf("cannot initialize metric fetcher: %v", err)
	}
	c := &metrics.Proxy{
		GetMetrics:           fetcher,
		AddEmptyDiskEnvelope: true,
	}
	setupAndStartMetricServer(loggr)

	var s *grpc.Server
	if cfg.TLS.CAPath != "" {
		loggr.Println("creating gRPC server with TLS")
		s = grpc.NewServer(
			grpc.UnaryInterceptor(requestTimer),
			grpc.Creds(cfg.TLS.Credentials("metric-proxy")),
		)
	} else {
		loggr.Println("creating gRPC server with no TLS")
		s = grpc.NewServer(
			grpc.UnaryInterceptor(requestTimer),
		)
	}
	logcache_v1.RegisterEgressServer(s, c)

	lis, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		loggr.Fatalf("failed to listen: %v", err)
	}
	panic(s.Serve(lis))
}

func setupAndStartMetricServer(loggr *log.Logger) {
	m := metricRegistry.NewRegistry(
		loggr,
		metricRegistry.WithPublicServer(
			9090,
		),
	)

	requestDurations = m.NewHistogram(
		"request_duration_seconds",
		"gPRC request duration distribution",
		[]float64{0.005, 2, 12},
	)
}

func requestTimer(ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler) (interface{}, error) {

	start := time.Now()

	h, err := handler(ctx, req)

	d := time.Now().Sub(start)

	requestDurations.Observe(d.Seconds())

	return h, err
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

	return func(guid string) (*v1beta1.PodMetricsList, error) {
		return c.MetricsV1beta1().PodMetricses(cfg.Namespace).List(v1.ListOptions{
			LabelSelector:  fmt.Sprintf("%s=%s", cfg.AppSelector, guid),
			TimeoutSeconds: &cfg.QueryTimeout,
		})
	}, nil
}
