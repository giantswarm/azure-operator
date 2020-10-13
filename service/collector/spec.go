package collector

import "github.com/prometheus/client_golang/prometheus"

type AzureAPIMetrics interface {
	GetCounter(opts prometheus.Opts) prometheus.Counter
	GetHistogram(opts prometheus.Opts) prometheus.Histogram
}
