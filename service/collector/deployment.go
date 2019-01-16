package collector

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest/to"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/prometheus/client_golang/prometheus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/controller/v4patch1/key"
	"github.com/giantswarm/azure-operator/service/credential"
)

const (
	statusCanceled  = "Canceled"
	statusFailed    = "Failed"
	statusRunning   = "Running"
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
	G8sClient versioned.Interface
	K8sClient kubernetes.Interface
	Logger    micrologger.Logger

	// EnvironmentName is the name of the Azure environment used to compute the
	// azure.Environment type. See also
	// https://godoc.org/github.com/Azure/go-autorest/autorest/azure#Environment.
	EnvironmentName string
}

type Deployment struct {
	g8sClient versioned.Interface
	k8sClient kubernetes.Interface
	logger    micrologger.Logger

	environmentName string
}

func NewDeployment(config DeploymentConfig) (*Deployment, error) {
	if config.G8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.G8sClient must not be empty", config)
	}
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if config.EnvironmentName == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.EnvironmentName must not be empty", config)
	}

	d := &Deployment{
		g8sClient: config.G8sClient,
		k8sClient: config.K8sClient,
		logger:    config.Logger,

		environmentName: config.EnvironmentName,
	}

	return d, nil
}

func (d *Deployment) Collect(ch chan<- prometheus.Metric) error {
	var crs []providerv1alpha1.AzureConfig
	{
		mark := ""
		page := 0
		for page == 0 || len(mark) > 0 {
			opts := metav1.ListOptions{
				Continue: mark,
			}
			list, err := d.g8sClient.ProviderV1alpha1().AzureConfigs("").List(opts)
			if err != nil {
				return microerror.Mask(err)
			}

			crs = append(crs, list.Items...)

			mark = list.Continue
			page++
		}
	}

	for _, cr := range crs {
		deploymentsClient, err := d.getDeploymentsClient(cr)
		if err != nil {
			return microerror.Mask(err)
		}

		r, err := deploymentsClient.ListByResourceGroup(context.Background(), key.ClusterID(cr), "", to.Int32Ptr(100))
		if err != nil {
			return microerror.Mask(err)
		}

		for r.NotDone() {
			for _, v := range r.Values() {
				ch <- prometheus.MustNewConstMetric(
					deploymentDesc,
					prometheus.GaugeValue,
					float64(matchedStringToInt(statusCanceled, *v.Properties.ProvisioningState)),
					key.ClusterID(cr),
					*v.Name,
					statusCanceled,
				)
				ch <- prometheus.MustNewConstMetric(
					deploymentDesc,
					prometheus.GaugeValue,
					float64(matchedStringToInt(statusFailed, *v.Properties.ProvisioningState)),
					key.ClusterID(cr),
					*v.Name,
					statusFailed,
				)
				ch <- prometheus.MustNewConstMetric(
					deploymentDesc,
					prometheus.GaugeValue,
					float64(matchedStringToInt(statusRunning, *v.Properties.ProvisioningState)),
					key.ClusterID(cr),
					*v.Name,
					statusRunning,
				)
				ch <- prometheus.MustNewConstMetric(
					deploymentDesc,
					prometheus.GaugeValue,
					float64(matchedStringToInt(statusSucceeded, *v.Properties.ProvisioningState)),
					key.ClusterID(cr),
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

	return nil
}

func (d *Deployment) Describe(ch chan<- *prometheus.Desc) error {
	ch <- deploymentDesc
	return nil
}

func (d *Deployment) getDeploymentsClient(cr providerv1alpha1.AzureConfig) (*resources.DeploymentsClient, error) {
	config, err := credential.GetAzureConfig(d.k8sClient, key.CredentialName(cr), key.CredentialNamespace(cr))
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
