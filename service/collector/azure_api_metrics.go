package collector

import (
	"sync"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/prometheus/client_golang/prometheus"
)

type Config struct {
	Logger micrologger.Logger
}

type AzureAPIMetricsCollector struct {
	logger micrologger.Logger

	counters   map[string]*prometheus.CounterVec
	histograms map[string]*prometheus.HistogramVec

	mutex *sync.Mutex
}

func NewAzureAPIMetricsCollector(config Config) (*AzureAPIMetricsCollector, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	c := AzureAPIMetricsCollector{
		logger: config.Logger,

		counters:   map[string]*prometheus.CounterVec{},
		histograms: map[string]*prometheus.HistogramVec{},
		mutex:      &sync.Mutex{},
	}

	return &c, nil
}

func (c *AzureAPIMetricsCollector) Describe(ch chan<- *prometheus.Desc) error {
	for _, c := range c.counters {
		c.Describe(ch)
	}

	for _, h := range c.histograms {
		h.Describe(ch)
	}

	return nil
}

func (c *AzureAPIMetricsCollector) Collect(ch chan<- prometheus.Metric) error {
	for _, c := range c.counters {
		c.Collect(ch)
	}

	for _, h := range c.histograms {
		h.Collect(ch)
	}

	return nil
}

func (c *AzureAPIMetricsCollector) GetCounterVec(opts prometheus.Opts, labelNames []string) *prometheus.CounterVec {
	k := opts.Namespace + "/" + opts.Name
	counter, exists := c.counters[k]
	if !exists {
		c.mutex.Lock()
		defer c.mutex.Unlock()

		counter, exists = c.counters[k]
		if !exists {
			counter = prometheus.NewCounterVec(prometheus.CounterOpts(opts), labelNames)
			c.counters[k] = counter
		}
	}

	return counter
}

func (c *AzureAPIMetricsCollector) GetHistogramVec(opts prometheus.Opts, labelNames []string) *prometheus.HistogramVec {
	k := opts.Namespace + "/" + opts.Name
	histogram, exists := c.histograms[k]
	if !exists {
		c.mutex.Lock()
		defer c.mutex.Unlock()

		histogram, exists = c.histograms[k]
		if !exists {
			o := prometheus.HistogramOpts{
				Namespace:   opts.Namespace,
				Name:        opts.Name,
				Help:        opts.Help,
				ConstLabels: opts.ConstLabels,
			}

			histogram = prometheus.NewHistogramVec(o, labelNames)
			c.histograms[k] = histogram
		}
	}

	return histogram
}
