package main

import (
	"bytes"
	"fmt"
	"io"
	"os"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
	"k8s.io/metrics/pkg/client/clientset/versioned"
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
	var timeout int64 = 5
	podMetrics, _ := metricsClient.MetricsV1beta1().PodMetricses(namespace).List(v1.ListOptions{
		LabelSelector:       "app=" + appSelector,
		TimeoutSeconds:      &timeout,
	})

	err := printMetrics(os.Stdout, podMetrics)
	if err != nil {
		panic(err)
	}
}

func printMetrics(w io.Writer, podMetrics *v1beta1.PodMetricsList) error {
	b := bytes.NewBuffer([]byte(""))
	for _, podMetric := range podMetrics.Items {
		b.WriteString(fmt.Sprintf("Metrics for pod: %s\n", podMetric.Name))
		for _, container := range podMetric.Containers {
			b.WriteString(fmt.Sprintf("\tcontainer: %s\n", container.Name))
			containerCPU := container.Usage.Cpu()
			containerMemory := container.Usage.Memory()
			b.WriteString(fmt.Sprintf("\tcpu: %v\n", containerCPU))
			b.WriteString(fmt.Sprintf("\tmemory: %v\n", containerMemory))
		}
	}
	_, err := b.WriteTo(w)
	return err
}
