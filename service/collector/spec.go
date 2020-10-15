package collector

import "github.com/prometheus/client_golang/prometheus"

type AzureAPIMetrics interface {
	GetCounterVec(opts prometheus.Opts, labelNames []string) *prometheus.CounterVec
	GetHistogramVec(opts prometheus.Opts, labelNames []string) *prometheus.HistogramVec
}
