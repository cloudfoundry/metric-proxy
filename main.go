package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"code.cloudfoundry.org/go-envstruct"
	"code.cloudfoundry.org/log-cache/pkg/rpc/logcache_v1"
	"code.cloudfoundry.org/metric-proxy/pkg/metrics"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
	"k8s.io/metrics/pkg/client/clientset/versioned"
)

var requestDurations = prometheus.NewHistogram(prometheus.HistogramOpts{
	Name:    "request_duration_seconds",
	Help:    "gPRC request duration distribution",
	Buckets: prometheus.ExponentialBuckets(0.005, 2, 12),
})

func init() {
	prometheus.MustRegister(requestDurations)
}

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

	startMetricsEndpoint()

	s := grpc.NewServer(
		grpc.UnaryInterceptor(requestTimer),
		grpc.Creds(cfg.TLS.Credentials("metric-proxy")),
	)
	logcache_v1.RegisterEgressServer(s, c)

	lis, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	panic(s.Serve(lis))
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

	return func(appGuid string) (*v1beta1.PodMetricsList, error) {
		return c.MetricsV1beta1().PodMetricses(cfg.Namespace).List(v1.ListOptions{
			LabelSelector:  fmt.Sprintf("%s=%s", cfg.AppSelector, appGuid),
			TimeoutSeconds: &cfg.QueryTimeout,
		})
	}, nil
}

func startMetricsEndpoint() {
	lis, err := net.Listen("tcp", ":9102")
	if err != nil {
		log.Printf("unable to start monitor endpoint: %s", err)
	}
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	log.Printf("starting monitor endpoint on http://%s/metrics\n", lis.Addr().String())
	go func() {
		err = http.Serve(lis, mux)
		log.Printf("error starting the monitor server: %s", err)
	}()
}
