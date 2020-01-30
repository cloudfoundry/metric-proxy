package metrics

import (
	"context"
	"fmt"
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
	t.Run("it returns envelopes with converted metrics", func(t *testing.T) {
		g := NewGomegaWithT(t)

		stop, err := startGRPCServer(
			newFakeFetcher(corev1.ResourceList{
				"cpu":    *resource.NewQuantity(42, "nanocores"),
			}))
		g.Expect(err).ToNot(HaveOccurred())
		defer stop()

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
		}))
	})

	t.Run("it returns an envelope for each metric", func(t *testing.T) {
		g := NewGomegaWithT(t)

		stop, err := startGRPCServer(
			newFakeFetcher(corev1.ResourceList{
				"metric1": *resource.NewQuantity(42, "metric1_format"),
				"metric2": *resource.NewQuantity(42, "metric2_format"),
				"metric3": *resource.NewQuantity(42, "metric3_format"),
				"metric4": *resource.NewQuantity(42, "metric4_format"),
			}))
		g.Expect(err).ToNot(HaveOccurred())
		defer stop()

		conn, err := grpc.Dial(":8080", grpc.WithInsecure())
		g.Expect(err).ToNot(HaveOccurred())
		defer conn.Close()

		client := logcache_v1.NewEgressClient(conn)
		resp, err := client.Read(context.Background(), &logcache_v1.ReadRequest{
			SourceId: "fake-source-1",
		})
		g.Expect(err).ToNot(HaveOccurred())

		g.Expect(resp.Envelopes.Batch).To(HaveLen(4))
		g.Expect(resp.Envelopes.Batch[0].SourceId).To(Equal("fake-source-1"))
	})

	t.Run("fails when there is an error fetching metrics", func(t *testing.T) {
		g := NewGomegaWithT(t)

		stop, err := startGRPCServer(
			newErrorFetcher("there is a fake error"),
		)
		g.Expect(err).ToNot(HaveOccurred())
		defer stop()

		conn, err := grpc.Dial(":8080", grpc.WithInsecure())
		g.Expect(err).ToNot(HaveOccurred())
		defer conn.Close()

		client := logcache_v1.NewEgressClient(conn)
		_, err = client.Read(context.Background(), &logcache_v1.ReadRequest{
			SourceId: "fake-source",
		})
		g.Expect(err).To(HaveOccurred())
	})
}

func startGRPCServer(f Fetcher) (stop func(), err error) {
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

func newFakeFetcher(resources corev1.ResourceList) Fetcher {
	return func() (*v1beta1.PodMetricsList, error) {
		return &v1beta1.PodMetricsList{
			TypeMeta: v1.TypeMeta{},
			ListMeta: v1.ListMeta{},
			Items: []v1beta1.PodMetrics{{
				Containers: []v1beta1.ContainerMetrics{{
					Name:  "test-container",
					Usage: resources,
				}},
			}},
		}, nil
	}
}

func newErrorFetcher(s string) Fetcher {
	return func() (*v1beta1.PodMetricsList, error) {
		return nil, fmt.Errorf(s)
	}
}
