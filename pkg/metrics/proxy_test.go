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
				"cpu": *resource.NewScaledQuantity(420000000, resource.Nano),
			}).GetMetrics, false)
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
				Unit:  "percentage",
				Value: 42.0,
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
			}).GetMetrics, false)
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
			newErrorFetcher("there is a fake error"), false,
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

	t.Run("it parses BinarySI format", func(t *testing.T) {
		g := NewGomegaWithT(t)

		stop, err := startGRPCServer(
			newFakeFetcher(corev1.ResourceList{
				"memory": *resource.NewQuantity(420000, "BinarySI"),
			}).GetMetrics, false)
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
			"memory": {
				Unit:  "bytes",
				Value: 420000,
			},
		}))
	})

	t.Run("it parses DecimalSI format", func(t *testing.T) {
		g := NewGomegaWithT(t)

		stop, err := startGRPCServer(
			newFakeFetcher(corev1.ResourceList{
				"cpu": *resource.NewScaledQuantity(500000000, resource.Nano),
			}).GetMetrics, false)
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
				Unit:  "percentage",
				Value: 50.0,
			},
		}))
	})

	t.Run("it adds empty disk gauges to each envelope list", func(t *testing.T) {
		g := NewGomegaWithT(t)

		stop, err := startGRPCServer(newFakeFetcher(corev1.ResourceList{}).GetMetrics, true)
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
			"disk": {
				Unit:  "bytes",
				Value: 0,
			},
		}))
	})

	t.Run("it calls GetMetrics with the GUID", func(t *testing.T) {
		g := NewGomegaWithT(t)

		f := newFakeFetcher(corev1.ResourceList{
			"cpu": *resource.NewScaledQuantity(420000000, resource.Nano),
		})

		stop, err := startGRPCServer(
			f.GetMetrics, false)
		g.Expect(err).ToNot(HaveOccurred())
		defer stop()

		conn, err := grpc.Dial(":8080", grpc.WithInsecure())
		g.Expect(err).ToNot(HaveOccurred())
		defer conn.Close()

		client := logcache_v1.NewEgressClient(conn)
		resp, err := client.Read(context.Background(), &logcache_v1.ReadRequest{
			SourceId: "fake-source-id",
		})
		g.Expect(err).ToNot(HaveOccurred())

		g.Eventually(f.guid).Should(Receive(Equal("fake-source-id")))
		g.Expect(resp.Envelopes.Batch).To(HaveLen(1))
	})

	t.Run("it sums cpu/mem metrics across containers in each pod", func(t *testing.T) {
		g := NewGomegaWithT(t)

		f := func(string) (*v1beta1.PodMetricsList, error) {
			return &v1beta1.PodMetricsList{
				TypeMeta: v1.TypeMeta{},
				ListMeta: v1.ListMeta{},
				Items: []v1beta1.PodMetrics{
					{
						Containers: []v1beta1.ContainerMetrics{
							{
								Name: "test-container-1",
								Usage: corev1.ResourceList{
									"cpu":    *resource.NewScaledQuantity(250000000, resource.Nano),
									"memory": *resource.NewQuantity(420000, "BinarySI"),
								},
							},
							{
								Name: "test-container-2",
								Usage: corev1.ResourceList{
									"cpu":    *resource.NewScaledQuantity(500000000, resource.Nano),
									"memory": *resource.NewQuantity(820000, "BinarySI"),
								},
							},
						},
					},
				},
			}, nil
		}

		stop, err := startGRPCServer(f, false)
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

		g.Expect(resp.Envelopes.Batch).To(HaveLen(2))
		results := map[string]*loggregator_v2.GaugeValue{}
		for _, e := range resp.Envelopes.Batch {
			for k, v := range e.GetGauge().Metrics {
				results[k] = v
			}
		}

		g.Expect(results).To(BeEquivalentTo(map[string]*loggregator_v2.GaugeValue{
			"cpu": {
				Unit:  "percentage",
				Value: 75.0,
			},
			"memory": {
				Unit:  "bytes",
				Value: 1240000,
			},
		}))
	})

	t.Run("it excludes platform containers from metric sums", func(t *testing.T) {
		g := NewGomegaWithT(t)

		f := func(string) (*v1beta1.PodMetricsList, error) {
			return &v1beta1.PodMetricsList{
				TypeMeta: v1.TypeMeta{},
				ListMeta: v1.ListMeta{},
				Items: []v1beta1.PodMetrics{
					{
						Containers: []v1beta1.ContainerMetrics{
							{
								Name: "istio-init",
								Usage: corev1.ResourceList{
									"cpu":    *resource.NewScaledQuantity(250000000, resource.Nano),
									"memory": *resource.NewQuantity(420000, "BinarySI"),
								},
							},
							{
								Name: "istio-proxy",
								Usage: corev1.ResourceList{
									"cpu":    *resource.NewScaledQuantity(500000000, resource.Nano),
									"memory": *resource.NewQuantity(820000, "BinarySI"),
								},
							},
							{
								Name: "test-container-1",
								Usage: corev1.ResourceList{
									"cpu":    *resource.NewScaledQuantity(250000000, resource.Nano),
									"memory": *resource.NewQuantity(420000, "BinarySI"),
								},
							},
							{
								Name: "test-container-2",
								Usage: corev1.ResourceList{
									"cpu":    *resource.NewScaledQuantity(500000000, resource.Nano),
									"memory": *resource.NewQuantity(820000, "BinarySI"),
								},
							},
						},
					},
				},
			}, nil
		}

		stop, err := startGRPCServer(f, false)
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

		g.Expect(resp.Envelopes.Batch).To(HaveLen(2))
		results := map[string]*loggregator_v2.GaugeValue{}
		for _, e := range resp.Envelopes.Batch {
			for k, v := range e.GetGauge().Metrics {
				results[k] = v
			}
		}

		g.Expect(results).To(BeEquivalentTo(map[string]*loggregator_v2.GaugeValue{
			"cpu": {
				Unit:  "percentage",
				Value: 75.0,
			},
			"memory": {
				Unit:  "bytes",
				Value: 1240000,
			},
		}))
	})
}

func startGRPCServer(f Fetcher, addEnvelopes bool) (stop func(), err error) {
	c := &Proxy{f, addEnvelopes}

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

type fakeFetcher struct {
	guid      chan string
	resources corev1.ResourceList
}

func newFakeFetcher(resources corev1.ResourceList) *fakeFetcher {
	return &fakeFetcher{
		guid:      make(chan string, 1),
		resources: resources,
	}
}

func (f *fakeFetcher) GetMetrics(guid string) (*v1beta1.PodMetricsList, error) {
	f.guid <- guid
	return &v1beta1.PodMetricsList{
		TypeMeta: v1.TypeMeta{},
		ListMeta: v1.ListMeta{},
		Items: []v1beta1.PodMetrics{{
			Containers: []v1beta1.ContainerMetrics{{
				Name:  "test-container",
				Usage: f.resources,
			}},
		}},
	}, nil
}

func newErrorFetcher(s string) Fetcher {
	return func(_ string) (*v1beta1.PodMetricsList, error) {
		return nil, fmt.Errorf(s)
	}
}
