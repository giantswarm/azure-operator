package collector

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/prometheus/client_golang/prometheus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/controller/v4/key"
	"github.com/giantswarm/azure-operator/service/credential"
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
	K8sClient kubernetes.Interface
	Logger    micrologger.Logger
	Watcher   func(opts metav1.ListOptions) (watch.Interface, error)

	// EnvironmentName is the name of the Azure environment used to compute the
	// azure.Environment type. See also
	// https://godoc.org/github.com/Azure/go-autorest/autorest/azure#Environment.
	EnvironmentName string
}

type Collector struct {
	k8sClient kubernetes.Interface
	logger    micrologger.Logger
	watcher   func(opts metav1.ListOptions) (watch.Interface, error)

	bootOnce sync.Once

	environmentName string
}

func New(config Config) (*Collector, error) {
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.Watcher == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Watcher must not be empty", config)
	}

	if config.EnvironmentName == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.EnvironmentName must not be empty", config)
	}

	c := &Collector{
		k8sClient: config.K8sClient,
		logger:    config.Logger,
		watcher:   config.Watcher,

		bootOnce: sync.Once{},

		environmentName: config.EnvironmentName,
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

			customObject, ok := event.Object.(*providerv1alpha1.AzureConfig)
			if !ok {
				c.logger.Log("level", "error", "message", "asserting AzureConfig struct failed")
				break
			}

			{
				n, s, err := c.getWorkerVMSSNameAndStatus(*customObject)
				if err != nil {
					c.logger.Log("level", "error", "message", "fetching worker VMSS name and status failed", "stack", fmt.Sprintf("%#v", err))
					return
				}

				ch <- prometheus.MustNewConstMetric(
					deploymentStatusDescription,
					prometheus.GaugeValue,
					float64(matchedStringToInt(statusCanceled, s)),
					customObject.GetName(),
					n,
					statusCanceled,
				)
				ch <- prometheus.MustNewConstMetric(
					deploymentStatusDescription,
					prometheus.GaugeValue,
					float64(matchedStringToInt(statusFailed, s)),
					customObject.GetName(),
					n,
					statusFailed,
				)
				ch <- prometheus.MustNewConstMetric(
					deploymentStatusDescription,
					prometheus.GaugeValue,
					float64(matchedStringToInt(statusSucceeded, s)),
					customObject.GetName(),
					n,
					statusSucceeded,
				)
			}
		case <-time.After(time.Second):
			c.logger.Log("level", "debug", "message", "finished collecting metrics")
			return
		}
	}
}

func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- deploymentStatusDescription
}

func (c *Collector) getWorkerVMSSNameAndStatus(customObject providerv1alpha1.AzureConfig) (string, string, error) {
	var scaleSetsClient *compute.VirtualMachineScaleSetsClient
	{
		config, err := credential.GetAzureConfig(c.k8sClient, key.CredentialName(customObject), key.CredentialNamespace(customObject))
		if err != nil {
			return "", "", microerror.Mask(err)
		}
		config.Cloud = c.environmentName

		azureClients, err := client.NewAzureClientSet(*config)
		if err != nil {
			return "", "", microerror.Mask(err)
		}
		scaleSetsClient = azureClients.VirtualMachineScaleSetsClient
	}

	var name string
	var status string
	{
		d, err := scaleSetsClient.Get(context.Background(), key.ResourceGroupName(customObject), key.WorkerVMSSName(customObject))
		if err != nil {
			return "", "", microerror.Mask(err)
		}

		name = key.WorkerVMSSName(customObject)
		status = *d.ProvisioningState
	}

	return name, status, nil
}

func matchedStringToInt(a, b string) int {
	if a == b {
		return 1
	}

	return 0
}
