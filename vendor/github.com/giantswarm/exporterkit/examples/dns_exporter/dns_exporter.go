package main

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/giantswarm/exporterkit"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	Namespace = "dns"
)

type Config struct {
	Hosts  []string
	Logger micrologger.Logger
}

type DNSCollector struct {
	hosts  []string
	logger micrologger.Logger

	total      *prometheus.Desc
	errorTotal *prometheus.Desc
	latency    *prometheus.Desc

	count      map[string]float64
	errorCount map[string]float64
}

func NewDNSCollector(config Config) (*DNSCollector, error) {
	if config.Hosts == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Hosts must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	dnsCollector := DNSCollector{
		hosts:  config.Hosts,
		logger: config.Logger,

		total: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "", "resolution_total"),
			"Total number of DNS resolutions.",
			[]string{"host"},
			nil,
		),
		errorTotal: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "", "resolution_error_total"),
			"Total number of DNS resolution errors.",
			[]string{"host"},
			nil,
		),
		latency: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "", "resolution_seconds"),
			"Time taken to resolve DNS.",
			[]string{"host"},
			nil,
		),

		count:      map[string]float64{},
		errorCount: map[string]float64{},
	}

	return &dnsCollector, nil
}

func (c *DNSCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.total
	ch <- c.errorTotal
	ch <- c.latency
}

func (c *DNSCollector) Collect(ch chan<- prometheus.Metric) {
	var wg sync.WaitGroup

	for _, host := range c.hosts {
		wg.Add(1)

		go func(host string) {
			defer wg.Done()

			start := time.Now()

			c.count[host]++

			_, err := net.LookupHost(host)
			if err != nil {
				c.logger.Log("level", "error", "message", "could not lookup host", "host", host, "stack", fmt.Sprintf("%#v", err))
				c.errorCount[host]++

				return
			}

			elapsed := time.Since(start)

			ch <- prometheus.MustNewConstMetric(c.total, prometheus.CounterValue, c.count[host], host)
			ch <- prometheus.MustNewConstMetric(c.errorTotal, prometheus.CounterValue, c.errorCount[host], host)
			ch <- prometheus.MustNewConstMetric(c.latency, prometheus.GaugeValue, elapsed.Seconds(), host)
		}(host)

		wg.Wait()
	}
}

func main() {
	var err error

	var logger micrologger.Logger
	{
		logger, err = micrologger.New(micrologger.Config{})
		if err != nil {
			panic(fmt.Sprintf("%#v\n", err))
		}
	}

	var dnsCollector prometheus.Collector
	{
		c := Config{
			Hosts: []string{
				"giantswarm.io",
				"google.com",
			},
			Logger: logger,
		}

		dnsCollector, err = NewDNSCollector(c)
		if err != nil {
			panic(fmt.Sprintf("%#v\n", err))
		}
	}

	var exporter *exporterkit.Exporter
	{
		c := exporterkit.Config{
			Collectors: []prometheus.Collector{
				dnsCollector,
			},
			Logger: logger,
		}

		exporter, err = exporterkit.New(c)
		if err != nil {
			panic(fmt.Sprintf("%#v\n", err))
		}
	}

	exporter.Run()
}
