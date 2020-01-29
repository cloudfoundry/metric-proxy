package metrics

import (
	"code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"
	"golang.org/x/net/context"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"

	"code.cloudfoundry.org/log-cache/pkg/rpc/logcache_v1"
)

type MetricsFetcher func() (*v1beta1.PodMetricsList, error)

func (m *Proxy) Read(_ context.Context, req *logcache_v1.ReadRequest) (*logcache_v1.ReadResponse, error) {
	var envelopes []*loggregator_v2.Envelope
	podMetrics, _ := m.GetMetrics()

	for _, podMetric := range podMetrics.Items {

		for _, container := range podMetric.Containers {
			var gauges = map[string]*loggregator_v2.GaugeValue{}

			for k, v := range container.Usage {
				if value, ok := v.AsInt64(); ok {
					gauges[string(k)] = &loggregator_v2.GaugeValue{
						Unit:  "",
						Value: float64(value),
					}
				}
			}

			envelopes = append(envelopes, &loggregator_v2.Envelope{
				Timestamp:  (req.GetStartTime() + req.EndTime) / 2,
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
			})
		}
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

type Proxy struct {
	GetMetrics MetricsFetcher

}
