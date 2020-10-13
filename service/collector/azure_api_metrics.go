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

	counters   map[string]prometheus.Counter
	histograms map[string]prometheus.Histogram

	mutex *sync.Mutex
}

func NewAzureAPIMetricsCollector(config Config) (*AzureAPIMetricsCollector, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	c := AzureAPIMetricsCollector{
		logger: config.Logger,

		counters:   map[string]prometheus.Counter{},
		histograms: map[string]prometheus.Histogram{},
		mutex:      &sync.Mutex{},
	}

	return &c, nil
}

func (c *AzureAPIMetricsCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, c := range c.counters {
		ch <- c.Desc()
	}

	for _, h := range c.histograms {
		ch <- h.Desc()
	}
}

func (c *AzureAPIMetricsCollector) Collect(ch chan<- prometheus.Metric) {
	for _, c := range c.counters {
		ch <- c
	}

	for _, h := range c.histograms {
		ch <- h
	}
}

func (c *AzureAPIMetricsCollector) GetCounter(opts prometheus.Opts) prometheus.Counter {
	k := opts.Namespace + "/" + opts.Name
	counter, exists := c.counters[k]
	if !exists {
		c.mutex.Lock()
		defer c.mutex.Unlock()

		counter, exists = c.counters[k]
		if !exists {
			counter = prometheus.NewCounter(prometheus.CounterOpts(opts))
			c.counters[k] = counter
		}
	}

	return counter
}

func (c *AzureAPIMetricsCollector) GetHistogram(opts prometheus.Opts) prometheus.Histogram {
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

			histogram = prometheus.NewHistogram(o)
			c.histograms[k] = histogram
		}
	}

	return histogram
}
