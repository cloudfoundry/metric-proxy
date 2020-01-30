package metrics

import (
	"time"

	"code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"
	"golang.org/x/net/context"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"

	"code.cloudfoundry.org/log-cache/pkg/rpc/logcache_v1"
)

type Fetcher func() (*v1beta1.PodMetricsList, error)

func (m *Proxy) Read(_ context.Context, req *logcache_v1.ReadRequest) (*logcache_v1.ReadResponse, error) {
	var envelopes []*loggregator_v2.Envelope
	podMetrics, err := m.GetMetrics()
	if err != nil {
		return nil, err
	}

	for _, podMetric := range podMetrics.Items {
		for _, container := range podMetric.Containers {
			for k, v := range container.Usage {
				envelopes = append(envelopes, m.createLoggregatorEnvelope(req, m.createGaugeMap(k, v)))
			}
		}
	}

	resp := &logcache_v1.ReadResponse{
		Envelopes: &loggregator_v2.EnvelopeBatch{
			Batch: envelopes,
		},
	}

	return resp, nil
}

func (m *Proxy) createLoggregatorEnvelope(
	req *logcache_v1.ReadRequest,
	gauges map[string]*loggregator_v2.GaugeValue,
) *loggregator_v2.Envelope {

	return &loggregator_v2.Envelope{
		Timestamp:  time.Now().UnixNano(),
		SourceId:   req.GetSourceId(),
		InstanceId: "0",
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
	var gauges = map[string]*loggregator_v2.GaugeValue{}

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
			Unit:  "nanocores",
			Value: value,
		}
	}

	return gauges
}

func (m *Proxy) Meta(context.Context, *logcache_v1.MetaRequest) (*logcache_v1.MetaResponse, error) {
	metaInfo := make(map[string]*logcache_v1.MetaInfo)

	return &logcache_v1.MetaResponse{
		Meta: metaInfo,
	}, nil
}

type Proxy struct {
	GetMetrics Fetcher
}
