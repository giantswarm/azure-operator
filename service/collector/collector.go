package collector

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest/to"
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

			customObject, err := key.ToCustomObject(event.Object)
			if err != nil {
				c.logger.Log("level", "error", "message", "asserting custom object failed", "stack", fmt.Sprintf("%#v", err))
				break
			}

			{
				deploymentsClient, err := c.getDeploymentsClient(customObject)
				if err != nil {
					c.logger.Log("level", "error", "message", "creating deployments client failed", "stack", fmt.Sprintf("%#v", err))
					return
				}

				r, err := deploymentsClient.ListByResourceGroup(context.Background(), key.ClusterID(customObject), "", to.Int32Ptr(100))
				if err != nil {
					c.logger.Log("level", "error", "message", "listing deployments failed", "stack", fmt.Sprintf("%#v", err))
					return
				}

				for r.NotDone() {
					for _, v := range r.Values() {
						c.logger.Log("level", "debug", "message", fmt.Sprintf("deployment %#q in resource group %#q has status %#q", *v.Name, key.ClusterID(customObject), *v.Properties.ProvisioningState))

						ch <- prometheus.MustNewConstMetric(
							deploymentStatusDescription,
							prometheus.GaugeValue,
							float64(matchedStringToInt(statusCanceled, *v.Properties.ProvisioningState)),
							key.ClusterID(customObject),
							*v.Name,
							statusCanceled,
						)
						ch <- prometheus.MustNewConstMetric(
							deploymentStatusDescription,
							prometheus.GaugeValue,
							float64(matchedStringToInt(statusFailed, *v.Properties.ProvisioningState)),
							key.ClusterID(customObject),
							*v.Name,
							statusFailed,
						)
						ch <- prometheus.MustNewConstMetric(
							deploymentStatusDescription,
							prometheus.GaugeValue,
							float64(matchedStringToInt(statusSucceeded, *v.Properties.ProvisioningState)),
							key.ClusterID(customObject),
							*v.Name,
							statusSucceeded,
						)
					}

					err := r.Next()
					if err != nil {
						c.logger.Log("level", "error", "message", "getting next page values failed", "stack", fmt.Sprintf("%#v", err))
						return
					}
				}
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

func (c *Collector) getDeploymentsClient(customObject providerv1alpha1.AzureConfig) (*resources.DeploymentsClient, error) {
	config, err := credential.GetAzureConfig(c.k8sClient, key.CredentialName(customObject), key.CredentialNamespace(customObject))
	if err != nil {
		return nil, microerror.Mask(err)
	}
	config.Cloud = c.environmentName

	azureClients, err := client.NewAzureClientSet(*config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return azureClients.DeploymentsClient, nil
}

func matchedStringToInt(a, b string) int {
	if a == b {
		return 1
	}

	return 0
}
