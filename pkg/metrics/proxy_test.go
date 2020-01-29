package metrics

import (
	"context"
	"net"
	"testing"

	"code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"
	"code.cloudfoundry.org/log-cache/pkg/rpc/logcache_v1"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
)

func TestMetricsProxyRead(t *testing.T) {
	g := NewGomegaWithT(t)

	stop, err := startGRPCServer(newFakeFetcher())
	g.Expect(err).ToNot(HaveOccurred())
	defer stop()

	t.Run("it returns metrics over gRPC", func(t *testing.T) {
		g := NewGomegaWithT(t)

		conn, err := grpc.Dial(":8080", grpc.WithInsecure())
		g.Expect(err).ToNot(HaveOccurred())
		defer conn.Close()

		client := logcache_v1.NewEgressClient(conn)
		resp, err := client.Read(context.Background(), &logcache_v1.ReadRequest{
			SourceId: "fake-source",
		})
		g.Expect(err).ToNot(HaveOccurred())

		g.Expect(resp.Envelopes.Batch).To(HaveLen(1))
		g.Expect(resp.Envelopes.Batch[0].SourceId).To(Equal("fake-source"))
		g.Expect(resp.Envelopes.Batch[0].GetGauge().Metrics).To(BeEquivalentTo(map[string]*loggregator_v2.GaugeValue{
			"cpu": {
				Unit:  "nanocores",
				Value: 42,
			},
			"memory": {
				Unit:  "MiB",
				Value: 42,
			},
		}))
	})
}

func startGRPCServer(f MetricsFetcher) (stop func(), err error) {
	c := &Proxy{f}

	s := grpc.NewServer()
	logcache_v1.RegisterEgressServer(s, c)

	lis, err := net.Listen("tcp", ":8080")
	if err != nil {
		return nil, err
	}

	go func() {
		err := s.Serve(lis)
		if err != nil {
			panic(err)
		}
	}()

	return s.GracefulStop, nil
}

func newFakeFetcher() func() (*v1beta1.PodMetricsList, error) {
	return func() (*v1beta1.PodMetricsList, error) {
		return &v1beta1.PodMetricsList{
			TypeMeta: v1.TypeMeta{},
			ListMeta: v1.ListMeta{},
			Items: []v1beta1.PodMetrics{{
				Containers: []v1beta1.ContainerMetrics{{
					Name: "test-container",
					Usage: corev1.ResourceList{
						"cpu":    *resource.NewQuantity(42, "nanocores"),
						"memory": *resource.NewQuantity(42, "MiB"),
					},
				}},
			}},
		}, nil
	}
}
