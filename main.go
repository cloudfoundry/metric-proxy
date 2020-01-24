package main

import (
	"fmt"
	"os"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/metrics/pkg/client/clientset/versioned"
)

func main() {
	apiServer := os.Getenv("APISERVER")
	token := os.Getenv("TOKEN")
	// caCertPath := os.Getenv("CACERT_PATH")
	namespace := "default"
	if value, ok := os.LookupEnv("NAMESPACE"); ok {
		namespace = value
	}

	restConfig := &rest.Config{
		Host:        apiServer,
		APIPath:     "/apis/metrics.k8s.io/v1beta1/pods",
		BearerToken: token,
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: true,
			// CAFile:   caCertPath,
		},
	}

	metricsClient, _ := versioned.NewForConfig(restConfig)
	// TODO: Use LabelSelector to select a particular pod
	podMetrics, _ := metricsClient.MetricsV1beta1().PodMetricses(namespace).List(v1.ListOptions{})

	for _, podMetric := range podMetrics.Items {
		for _, container := range podMetric.Containers {
			fmt.Println("Metrics for container: ", container.Name)
			containerCPU := container.Usage.Cpu()
			containerMemory := container.Usage.Memory()
			fmt.Println("cpu: ", containerCPU)
			fmt.Println("memory: ", containerMemory)
		}
	}
}
