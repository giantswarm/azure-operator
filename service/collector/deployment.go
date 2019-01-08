package collector

import (
	"context"
	"fmt"
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
	"github.com/giantswarm/azure-operator/service/controller/v4patch1/key"
	"github.com/giantswarm/azure-operator/service/credential"
)

const (
	statusCanceled  = "Canceled"
	statusDeploying = "Deploying"
	statusFailed    = "Failed"
	statusUpdating  = "Updating"
	statusSucceeded = "Succeeded"
)

var (
	deploymentDesc *prometheus.Desc = prometheus.NewDesc(
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

type DeploymentConfig struct {
	K8sClient kubernetes.Interface
	Logger    micrologger.Logger
	Watcher   func(opts metav1.ListOptions) (watch.Interface, error)

	// EnvironmentName is the name of the Azure environment used to compute the
	// azure.Environment type. See also
	// https://godoc.org/github.com/Azure/go-autorest/autorest/azure#Environment.
	EnvironmentName string
}

type Deployment struct {
	k8sClient kubernetes.Interface
	logger    micrologger.Logger
	watcher   func(opts metav1.ListOptions) (watch.Interface, error)

	environmentName string
}

func NewDeployment(config DeploymentConfig) (*Deployment, error) {
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

	d := &Deployment{
		k8sClient: config.K8sClient,
		logger:    config.Logger,
		watcher:   config.Watcher,

		environmentName: config.EnvironmentName,
	}

	return d, nil
}

func (d *Deployment) Collect(ch chan<- prometheus.Metric) error {
	watcher, err := d.watcher(metav1.ListOptions{})
	if err != nil {
		return microerror.Mask(err)
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
				return microerror.Mask(err)
			}

			{
				deploymentsClient, err := d.getDeploymentsClient(customObject)
				if err != nil {
					return microerror.Mask(err)
				}

				r, err := deploymentsClient.ListByResourceGroup(context.Background(), key.ClusterID(customObject), "", to.Int32Ptr(100))
				if err != nil {
					return microerror.Mask(err)
				}

				for r.NotDone() {
					for _, v := range r.Values() {
						fmt.Printf("*v.Properties.ProvisioningState: %#v\n", *v.Properties.ProvisioningState)

						ch <- prometheus.MustNewConstMetric(
							deploymentDesc,
							prometheus.GaugeValue,
							float64(matchedStringToInt(statusCanceled, *v.Properties.ProvisioningState)),
							key.ClusterID(customObject),
							*v.Name,
							statusCanceled,
						)
						ch <- prometheus.MustNewConstMetric(
							deploymentDesc,
							prometheus.GaugeValue,
							float64(matchedStringToInt(statusDeploying, *v.Properties.ProvisioningState)),
							key.ClusterID(customObject),
							*v.Name,
							statusDeploying,
						)
						ch <- prometheus.MustNewConstMetric(
							deploymentDesc,
							prometheus.GaugeValue,
							float64(matchedStringToInt(statusFailed, *v.Properties.ProvisioningState)),
							key.ClusterID(customObject),
							*v.Name,
							statusFailed,
						)
						ch <- prometheus.MustNewConstMetric(
							deploymentDesc,
							prometheus.GaugeValue,
							float64(matchedStringToInt(statusUpdating, *v.Properties.ProvisioningState)),
							key.ClusterID(customObject),
							*v.Name,
							statusUpdating,
						)
						ch <- prometheus.MustNewConstMetric(
							deploymentDesc,
							prometheus.GaugeValue,
							float64(matchedStringToInt(statusSucceeded, *v.Properties.ProvisioningState)),
							key.ClusterID(customObject),
							*v.Name,
							statusSucceeded,
						)
					}

					err := r.Next()
					if err != nil {
						return microerror.Mask(err)
					}
				}
			}
		case <-time.After(time.Second):
			return nil
		}
	}
}

func (d *Deployment) Describe(ch chan<- *prometheus.Desc) error {
	ch <- deploymentDesc
	return nil
}

func (d *Deployment) getDeploymentsClient(customObject providerv1alpha1.AzureConfig) (*resources.DeploymentsClient, error) {
	config, err := credential.GetAzureConfig(d.k8sClient, key.CredentialName(customObject), key.CredentialNamespace(customObject))
	if err != nil {
		return nil, microerror.Mask(err)
	}
	config.Cloud = d.environmentName

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
