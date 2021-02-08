package metrics

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"
	"golang.org/x/net/context"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"

	"code.cloudfoundry.org/log-cache/pkg/rpc/logcache_v1"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

//counterfeiter:generate . DiskUsageFetcher

type DiskUsageFetcher interface {
	DiskUsage(podName string) (int64, error)
}

type MetricsFetcherFn func(guid string) (*v1beta1.PodMetricsList, error)

type Proxy struct {
	logger           *log.Logger
	metricsFetcherFn MetricsFetcherFn
	diskUsageFetcher DiskUsageFetcher
}

func NewProxy(logger *log.Logger, metricsFetcherFn MetricsFetcherFn, diskUsageFetcher DiskUsageFetcher) *Proxy {
	return &Proxy{
		logger:           logger,
		metricsFetcherFn: metricsFetcherFn,
		diskUsageFetcher: diskUsageFetcher,
	}
}

func (m *Proxy) Read(_ context.Context, req *logcache_v1.ReadRequest) (*logcache_v1.ReadResponse, error) {
	var envelopes []*loggregator_v2.Envelope

	podMetrics, err := m.metricsFetcherFn(req.SourceId)
	if err != nil {
		m.logger.Printf("failed to get metrics: %v", err)
		return nil, err
	}

	for _, podMetric := range podMetrics.Items {
		metrics := aggregateContainerMetrics(podMetric.Containers)

		for k, v := range metrics {
			envelopes = append(envelopes,
				m.createLoggregatorEnvelope(
					req,
					m.createGaugeMap(v1.ResourceName(k), v),
					getInstanceID(podMetric),
				),
			)
		}

		diskEnvelope, err := m.createDiskEnvelope(req, podMetric)
		if err != nil {
			return nil, fmt.Errorf("failed getting disk usage: %w", err)
		}
		envelopes = append(envelopes, diskEnvelope)
	}

	resp := &logcache_v1.ReadResponse{
		Envelopes: &loggregator_v2.EnvelopeBatch{
			Batch: envelopes,
		},
	}

	return resp, nil
}

func (m *Proxy) Meta(context.Context, *logcache_v1.MetaRequest) (*logcache_v1.MetaResponse, error) {
	metaInfo := make(map[string]*logcache_v1.MetaInfo)

	return &logcache_v1.MetaResponse{
		Meta: metaInfo,
	}, nil
}

func aggregateContainerMetrics(containers []v1beta1.ContainerMetrics) map[string]resource.Quantity {
	metrics := map[string]resource.Quantity{}

	for _, container := range containers {
		if isIstio(container.Name) {
			continue
		}
		for k, v := range container.Usage {
			if value, ok := metrics[string(k)]; ok {
				value.Add(v)
				metrics[string(k)] = value
			} else {
				metrics[string(k)] = v
			}
		}
	}

	return metrics
}

func isIstio(podName string) bool {
	b, _ := regexp.MatchString("istio\\-.*", podName)
	return b
}

func (m *Proxy) createDiskEnvelope(req *logcache_v1.ReadRequest, podMetric v1beta1.PodMetrics) (*loggregator_v2.Envelope, error) {
	instanceID := getInstanceID(podMetric)

	podDiskUsage, err := m.diskUsageFetcher.DiskUsage(podMetric.Name)
	if err != nil {
		m.logger.Printf("error fetching disk usage: %v", err)
		return nil, err
	}

	return m.createLoggregatorEnvelope(
		req,
		m.createGaugeMap(
			"disk", *resource.NewQuantity(podDiskUsage, "BinarySI"),
		),
		instanceID,
	), nil
}

func (m *Proxy) createLoggregatorEnvelope(
	req *logcache_v1.ReadRequest,
	gauges map[string]*loggregator_v2.GaugeValue,
	instanceID string,
) *loggregator_v2.Envelope {
	return &loggregator_v2.Envelope{
		Timestamp:  time.Now().UnixNano(),
		SourceId:   req.GetSourceId(),
		InstanceId: instanceID,
		Tags: map[string]string{
			"process_id": req.GetSourceId(),
			"origin":     "rep",
		},
		Message: &loggregator_v2.Envelope_Gauge{
			Gauge: &loggregator_v2.Gauge{
				Metrics: gauges,
			},
		},
	}
}

func (m *Proxy) createGaugeMap(k v1.ResourceName, v resource.Quantity) map[string]*loggregator_v2.GaugeValue {
	gauges := map[string]*loggregator_v2.GaugeValue{}

	switch v.Format {
	case "BinarySI":
		if value, ok := v.AsInt64(); ok {
			gauges[string(k)] = &loggregator_v2.GaugeValue{
				Unit:  "bytes",
				Value: float64(value),
			}
		}
	case "DecimalSI":
		value := float64(v.ScaledValue(resource.Nano))
		gauges[string(k)] = &loggregator_v2.GaugeValue{
			Unit:  "percentage",
			Value: value / 1e7,
		}
	}

	return gauges
}

func getInstanceID(podMetric v1beta1.PodMetrics) string {
	s := strings.Split(podMetric.Name, "-")
	return s[len(s)-1]
}
