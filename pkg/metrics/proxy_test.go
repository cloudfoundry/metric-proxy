package metrics_test

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"testing"

	"code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"
	"code.cloudfoundry.org/log-cache/pkg/rpc/logcache_v1"
	"code.cloudfoundry.org/metric-proxy/pkg/metrics"
	"code.cloudfoundry.org/metric-proxy/pkg/metrics/metricsfakes"
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

		fakeDiskUsageFetcher := new(metricsfakes.FakeDiskUsageFetcher)
		f := newFakeMetricsFetcher(corev1.ResourceList{
			"cpu": *resource.NewScaledQuantity(420000000, resource.Nano),
		})
		stop, err := startGRPCServer(f.GetMetrics, fakeDiskUsageFetcher)
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

		fakeDiskUsageFetcher := new(metricsfakes.FakeDiskUsageFetcher)
		f := newFakeMetricsFetcher(corev1.ResourceList{
			"metric1": *resource.NewQuantity(42, "metric1_format"),
			"metric2": *resource.NewQuantity(42, "metric2_format"),
			"metric3": *resource.NewQuantity(42, "metric3_format"),
			"metric4": *resource.NewQuantity(42, "metric4_format"),
		})
		stop, err := startGRPCServer(f.GetMetrics, fakeDiskUsageFetcher)
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

		g.Expect(resp.Envelopes.Batch).To(HaveLen(5))
		g.Expect(resp.Envelopes.Batch[0].SourceId).To(Equal("fake-source-1"))
	})

	t.Run("fails when there is an error fetching metrics", func(t *testing.T) {
		g := NewGomegaWithT(t)

		fakeDiskUsageFetcher := new(metricsfakes.FakeDiskUsageFetcher)
		f := newErrorFetcher("there is a fake error")
		stop, err := startGRPCServer(f, fakeDiskUsageFetcher)
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

	t.Run("fails when there is an error fetching disk usage", func(t *testing.T) {
		g := NewGomegaWithT(t)

		fakeDiskUsageFetcher := new(metricsfakes.FakeDiskUsageFetcher)
		fakeDiskUsageFetcher.DiskUsageReturns(0, errors.New("k8s problem"))
		f := newFakeMetricsFetcher(corev1.ResourceList{
			"cpu": *resource.NewScaledQuantity(420000000, resource.Nano),
		})
		stop, err := startGRPCServer(f.GetMetrics, fakeDiskUsageFetcher)
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

		fakeDiskUsageFetcher := new(metricsfakes.FakeDiskUsageFetcher)
		f := newFakeMetricsFetcher(corev1.ResourceList{
			"memory": *resource.NewQuantity(420000, "BinarySI"),
		})
		stop, err := startGRPCServer(f.GetMetrics, fakeDiskUsageFetcher)
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

		fakeDiskUsageFetcher := new(metricsfakes.FakeDiskUsageFetcher)
		f := newFakeMetricsFetcher(corev1.ResourceList{
			"cpu": *resource.NewScaledQuantity(500000000, resource.Nano),
		})
		stop, err := startGRPCServer(f.GetMetrics, fakeDiskUsageFetcher)
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
		g.Expect(resp.Envelopes.Batch[0].SourceId).To(Equal("fake-source"))
		g.Expect(resp.Envelopes.Batch[0].GetGauge().Metrics).To(BeEquivalentTo(map[string]*loggregator_v2.GaugeValue{
			"cpu": {
				Unit:  "percentage",
				Value: 50.0,
			},
		}))
	})

	t.Run("it adds disk gauges to each envelope list", func(t *testing.T) {
		g := NewGomegaWithT(t)

		fakeDiskUsageFetcher := new(metricsfakes.FakeDiskUsageFetcher)
		fakeDiskUsageFetcher.DiskUsageReturns(300, nil)
		f := newFakeMetricsFetcher(corev1.ResourceList{})
		stop, err := startGRPCServer(f.GetMetrics, fakeDiskUsageFetcher)
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
				Value: 300,
			},
		}))
	})

	t.Run("it calls GetMetrics with the process GUID", func(t *testing.T) {
		g := NewGomegaWithT(t)

		fakeDiskUsageFetcher := new(metricsfakes.FakeDiskUsageFetcher)
		f := newFakeMetricsFetcher(corev1.ResourceList{
			"cpu": *resource.NewScaledQuantity(420000000, resource.Nano),
		})
		stop, err := startGRPCServer(f.GetMetrics, fakeDiskUsageFetcher)
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

		g.Eventually(f.processGUID).Should(Receive(Equal("fake-source-id")))
		g.Expect(resp.Envelopes.Batch).To(HaveLen(2))
	})

	t.Run("it sums cpu/mem metrics across containers in each pod", func(t *testing.T) {
		g := NewGomegaWithT(t)

		fakeDiskUsageFetcher := new(metricsfakes.FakeDiskUsageFetcher)
		fakeDiskUsageFetcher.DiskUsageReturns(1234, nil)
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
		stop, err := startGRPCServer(f, fakeDiskUsageFetcher)

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

		g.Expect(resp.Envelopes.Batch).To(HaveLen(3))
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
			"disk": {
				Unit:  "bytes",
				Value: 1234,
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

		fakeDiskUsageFetcher := new(metricsfakes.FakeDiskUsageFetcher)
		fakeDiskUsageFetcher.DiskUsageReturns(1234, nil)

		stop, err := startGRPCServer(f, fakeDiskUsageFetcher)
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

		g.Expect(resp.Envelopes.Batch).To(HaveLen(3))
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
			"disk": {
				Unit:  "bytes",
				Value: 1234,
			},
		}))
	})

	t.Run("it returns metrics with InstanceId based on pod name", func(t *testing.T) {
		g := NewGomegaWithT(t)
		fakeDiskUsageFetcher := new(metricsfakes.FakeDiskUsageFetcher)
		f := newFakeMetricsFetcher(corev1.ResourceList{
			"cpu": *resource.NewScaledQuantity(420000000, resource.Nano),
		})
		f.appCount = 2

		stop, err := startGRPCServer(f.GetMetrics, fakeDiskUsageFetcher)
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

		g.Expect(resp.Envelopes.Batch[0].InstanceId).To(Equal("0"))
		g.Expect(resp.Envelopes.Batch[1].InstanceId).To(Equal("0"))
		g.Expect(resp.Envelopes.Batch[2].InstanceId).To(Equal("1"))
		g.Expect(resp.Envelopes.Batch[3].InstanceId).To(Equal("1"))
	})
}

func startGRPCServer(f metrics.MetricsFetcherFn, d metrics.DiskUsageFetcher) (stop func(), err error) {
	logger := log.New(os.Stderr, "", log.LstdFlags)
	c := metrics.NewProxy(logger, f, d)

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

type fakeMetricsFetcher struct {
	appCount    int
	processGUID chan string
	podName     chan string
	resources   corev1.ResourceList
}

func newFakeMetricsFetcher(resources corev1.ResourceList) *fakeMetricsFetcher {
	return &fakeMetricsFetcher{
		processGUID: make(chan string, 1),
		podName:     make(chan string, 1),
		resources:   resources,
		appCount:    1,
	}
}

func (f *fakeMetricsFetcher) GetMetrics(processGUID string) (*v1beta1.PodMetricsList, error) {
	f.processGUID <- processGUID
	return &v1beta1.PodMetricsList{
		TypeMeta: v1.TypeMeta{},
		ListMeta: v1.ListMeta{},
		Items:    f.podMetrics(),
	}, nil
}

func (f *fakeMetricsFetcher) podMetrics() []v1beta1.PodMetrics {
	m := make([]v1beta1.PodMetrics, 0)
	for i := 0; i < f.appCount; i++ {
		m = append(m, v1beta1.PodMetrics{
			ObjectMeta: v1.ObjectMeta{
				Name: fmt.Sprintf("test-app-%v", i),
			},
			Containers: []v1beta1.ContainerMetrics{{
				Name:  "test-app",
				Usage: f.resources,
			}},
		})
	}
	return m
}

func newErrorFetcher(s string) metrics.MetricsFetcherFn {
	return func(_ string) (*v1beta1.PodMetricsList, error) {
		return nil, fmt.Errorf(s)
	}
}
