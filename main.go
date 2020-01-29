package main

import (
	"log"
	"net"
	"os"

	"google.golang.org/grpc"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
	"k8s.io/metrics/pkg/client/clientset/versioned"

	"code.cloudfoundry.org/log-cache/pkg/rpc/logcache_v1"
	"google.golang.org/grpc/reflection"

	"github.com/loggregator/metric-scraper/pkg/metrics"
)

func main() {
	apiServer := os.Getenv("APISERVER")
	token := os.Getenv("TOKEN")
	appSelector := os.Getenv("APP_SELECTOR")
	// caCertPath := os.Getenv("CACERT_PATH")
	namespace := "default"
	if value, ok := os.LookupEnv("NAMESPACE"); ok {
		namespace = value
	}

	// TODO: handle errors
	fetcher, _ := createMetricsFetcher(apiServer, token, namespace, appSelector)
	c := &metrics.Proxy{
		GetMetrics: fetcher,
	}

	s := grpc.NewServer()
	logcache_v1.RegisterEgressServer(s, c)
	reflection.Register(s)

	lis, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	panic(s.Serve(lis))
}

func createMetricsFetcher(apiServer, token, namespace, appSelector string) (metrics.MetricsFetcher, error) {
	restConfig := &rest.Config{
		Host:        apiServer,
		APIPath:     "/apis/metrics.k8s.io/v1beta1/pods",
		BearerToken: token,
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: true,
		},
	}

	c, err := versioned.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	return func() (*v1beta1.PodMetricsList, error) {
		var timeout int64 = 5
		return c.MetricsV1beta1().PodMetricses(namespace).List(v1.ListOptions{
			LabelSelector:  "app=" + appSelector,
			TimeoutSeconds: &timeout,
		})
	}, nil
}
