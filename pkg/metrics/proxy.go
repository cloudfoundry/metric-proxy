package metrics

import (
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

type NodeDiskUsage struct {
	Pods []PodDiskUsage `json:"pods"`
}

type PodDiskUsage struct {
	PodRef     PodRef               `json:"podRef"`
	Containers []ContainerDiskUsage `json:"containers"`
}

type PodRef struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type ContainerDiskUsage struct {
	Name   string    `json:"name"`
	RootFS DiskUsage `json:"rootfs"`
	Logs   DiskUsage `json:"logs"`
}

type DiskUsage struct {
	AvailableBytes int64 `json:"availableBytes"`
	CapacityBytes  int64 `json:"capacityBytes"`
	UsedBytes      int64 `json:"usedBytes"`
}

type (
	Fetcher          func(guid string) (*v1beta1.PodMetricsList, error)
	DiskUsageFetcher func(guid string) (PodDiskUsage, error)
)

type Proxy struct {
	GetMetrics           Fetcher
	GetDiskUsage         DiskUsageFetcher
	AddEmptyDiskEnvelope bool
}

func (m *Proxy) Read(_ context.Context, req *logcache_v1.ReadRequest) (*logcache_v1.ReadResponse, error) {
	var envelopes []*loggregator_v2.Envelope

	podMetrics, err := m.GetMetrics(req.SourceId)
	if err != nil {
		return nil, err
	}

	for _, podMetric := range podMetrics.Items {
		metrics := aggregateContainerMetrics(podMetric.Containers)

		for k, v := range metrics {
			envelopes = append(envelopes,
				m.createLoggregatorEnvelope(
					req,
					m.createGaugeMap(v1.ResourceName(k), v),
					getInstanceId(podMetric),
				),
			)
		}

		envelopes = append(envelopes, m.createDiskEnvelope(req, podMetric))
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

func (m *Proxy) createEmptyDiskEnvelope(req *logcache_v1.ReadRequest, instanceId string) *loggregator_v2.Envelope {
	return m.createLoggregatorEnvelope(
		req,
		m.createGaugeMap(
			"disk", *resource.NewQuantity(0, "BinarySI"),
		),
		instanceId,
	)
}

func (m *Proxy) createDiskEnvelope(req *logcache_v1.ReadRequest, podMetric v1beta1.PodMetrics) *loggregator_v2.Envelope {
	podDiskUsage, err := m.GetDiskUsage(podMetric.Name)
	instanceID := getInstanceId(podMetric)

	if err != nil {
		return m.createLoggregatorEnvelope(
			req,
			m.createGaugeMap(
				"disk", *resource.NewQuantity(0, "BinarySI"),
			),
			instanceID,
		)
	}

	var usedBytes int64
	for _, cdu := range podDiskUsage.Containers {
		if !isIstio(cdu.Name) {
			usedBytes += cdu.RootFS.UsedBytes + cdu.Logs.UsedBytes
		}
	}

	return m.createLoggregatorEnvelope(
		req,
		m.createGaugeMap(
			"disk", *resource.NewQuantity(usedBytes, "BinarySI"),
		),
		instanceID,
	)
}

func (m *Proxy) createLoggregatorEnvelope(
	req *logcache_v1.ReadRequest,
	gauges map[string]*loggregator_v2.GaugeValue,
	instanceId string,
) *loggregator_v2.Envelope {

	return &loggregator_v2.Envelope{
		Timestamp:  time.Now().UnixNano(),
		SourceId:   req.GetSourceId(),
		InstanceId: instanceId,
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

func getInstanceId(podMetric v1beta1.PodMetrics) string {
	s := strings.Split(podMetric.Name, "-")
	return s[len(s)-1]
}
