package collector

import (
	"context"
	"fmt"
	"sync"
	"time"

	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/prometheus/client_golang/prometheus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

const (
	statusCanceled  = "Canceled"
	statusFailed    = "Failed"
	statusSucceeded = "Succeeded"
)

var (
	deploymentStatusDescription *prometheus.Desc = prometheus.NewDesc(
		prometheus.BuildFQName("azure_operator", "deployment", "status"),
		"Cluster status condition as provided by the CR status.",
		[]string{
			"cluster_id",
			"deployment_name",
			"status",
		},
		nil,
	)
)

type Config struct {
	Logger  micrologger.Logger
	Watcher func(opts metav1.ListOptions) (watch.Interface, error)
}

type Collector struct {
	logger  micrologger.Logger
	watcher func(opts metav1.ListOptions) (watch.Interface, error)

	bootOnce sync.Once
}

func New(config Config) (*Collector, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.Watcher == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Watcher must not be empty", config)
	}

	c := &Collector{
		logger:  config.Logger,
		watcher: config.Watcher,

		bootOnce: sync.Once{},
	}

	return c, nil
}

func (c *Collector) Boot(ctx context.Context) {
	c.bootOnce.Do(func() {
		c.logger.LogCtx(ctx, "level", "debug", "message", "registering collector")

		err := prometheus.Register(prometheus.Collector(c))
		if IsAlreadyRegisteredError(err) {
			c.logger.LogCtx(ctx, "level", "debug", "message", "collector already registered")
		} else if err != nil {
			c.logger.Log("level", "error", "message", "registering collector failed", "stack", fmt.Sprintf("%#v", err))
		} else {
			c.logger.LogCtx(ctx, "level", "debug", "message", "registered collector")
		}
	})
}

func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	c.logger.Log("level", "debug", "message", "start collecting metrics")

	watcher, err := c.watcher(metav1.ListOptions{})
	if err != nil {
		c.logger.Log("level", "error", "message", "watching CRs failed", "stack", fmt.Sprintf("%#v", err))
		return
	}
	defer watcher.Stop()

	for {
		select {
		case event, ok := <-watcher.ResultChan():
			if !ok {
				continue
			}

			a, ok := event.Object.(*providerv1alpha1.AzureConfig)
			if !ok {
				c.logger.Log("level", "error", "message", "asserting AzureConfig struct failed")
				break
			}

			// TODO create azure clients
			// TODO fetch deployment status
			n := "worker_vmss"
			s := ""
			// TODO

			ch <- prometheus.MustNewConstMetric(
				deploymentStatusDescription,
				prometheus.GaugeValue,
				float64(matchedStringToInt(statusCanceled, s)),
				a.GetName(),
				n,
				statusCanceled,
			)
			ch <- prometheus.MustNewConstMetric(
				deploymentStatusDescription,
				prometheus.GaugeValue,
				float64(matchedStringToInt(statusFailed, s)),
				a.GetName(),
				n,
				statusFailed,
			)
			ch <- prometheus.MustNewConstMetric(
				deploymentStatusDescription,
				prometheus.GaugeValue,
				float64(matchedStringToInt(statusSucceeded, s)),
				a.GetName(),
				n,
				statusSucceeded,
			)
		case <-time.After(time.Second):
			c.logger.Log("level", "debug", "message", "finished collecting metrics")
			return
		}
	}
}

func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- deploymentStatusDescription
}

func matchedStringToInt(a, b string) int {
	if a == b {
		return 1
	}

	return 0
}
