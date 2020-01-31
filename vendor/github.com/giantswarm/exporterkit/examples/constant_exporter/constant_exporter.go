package main

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/giantswarm/exporterkit"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
)

type Config struct {
	Logger micrologger.Logger

	Constant float64
}

type ConstantCollector struct {
	logger micrologger.Logger

	constant float64

	constantDesc *prometheus.Desc
}

func NewConstantCollector(config Config) (*ConstantCollector, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	constantCollector := ConstantCollector{
		logger: config.Logger,

		constant: config.Constant,

		constantDesc: prometheus.NewDesc(
			prometheus.BuildFQName("constant_exporter", "", "constant"),
			"The constant.",
			nil,
			nil,
		),
	}

	return &constantCollector, nil
}

func (c *ConstantCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.constantDesc
}

func (c *ConstantCollector) Collect(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(c.constantDesc, prometheus.GaugeValue, c.constant)
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

	var constantCollector prometheus.Collector
	{
		c := Config{
			Logger: logger,

			Constant: float64(1),
		}

		constantCollector, err = NewConstantCollector(c)
		if err != nil {
			panic(fmt.Sprintf("%#v\n", err))
		}
	}

	var exporter *exporterkit.Exporter
	{
		c := exporterkit.Config{
			Collectors: []prometheus.Collector{
				constantCollector,
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
