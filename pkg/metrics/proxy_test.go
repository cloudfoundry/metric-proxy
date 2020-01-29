package metrics

import (
	"net"
	"testing"
	"code.cloudfoundry.org/log-cache/pkg/rpc/logcache_v1"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
)

func TestMetricsClientGRPC(t *testing.T) {
	g := NewGomegaWithT(t)
	c := &Proxy{
		newFakeFetcher(),
	}

	s := grpc.NewServer()
	logcache_v1.RegisterEgressServer(s, c)
	reflection.Register(s)


	lis, err := net.Listen("tcp", ":8080")
	g.Expect(err).ToNot(HaveOccurred())

	defer s.GracefulStop()
	go func() {
		err := s.Serve(lis)
		g.Expect(err).ToNot(HaveOccurred())
	}()

	t.Run("it works", func(t *testing.T) {
		g := NewGomegaWithT(t)

		_, err := grpc.Dial(":8080", grpc.WithInsecure())
		g.Expect(err).ToNot(HaveOccurred())
	})
}

func newFakeFetcher() func() (*v1beta1.PodMetricsList, error) {
	return func() (*v1beta1.PodMetricsList, error) {
		return &v1beta1.PodMetricsList{
			TypeMeta: v1.TypeMeta{},
			ListMeta: v1.ListMeta{},
			Items:    []v1beta1.PodMetrics{{
				Containers: []v1beta1.ContainerMetrics{{
					Name: "test-container",
					Usage: corev1.ResourceList{
						"cpu":    *resource.NewQuantity(42, "cpu"),
						"memory": *resource.NewQuantity(42, "mem"),
					},
				}},
			}},
		}, nil
	}
}