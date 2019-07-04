package collector

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/prometheus/client_golang/prometheus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/controller/v9/key"
	"github.com/giantswarm/azure-operator/service/credential"
)

var (
	usageCurrentDesc *prometheus.Desc = prometheus.NewDesc(
		prometheus.BuildFQName("azure_operator", "usage", "current"),
		"Current usage of specific Quotas as defined by Azure.",
		[]string{
			"name",
		},
		nil,
	)
	usageLimitDesc *prometheus.Desc = prometheus.NewDesc(
		prometheus.BuildFQName("azure_operator", "usage", "limit"),
		"Usage limit of specific Quotas as defined by Azure.",
		[]string{
			"name",
		},
		nil,
	)
)

type UsageConfig struct {
	G8sClient versioned.Interface
	K8sClient kubernetes.Interface
	Logger    micrologger.Logger

	// EnvironmentName is the name of the Azure environment used to compute the
	// azure.Environment type. See also
	// https://godoc.org/github.com/Azure/go-autorest/autorest/azure#Environment.
	EnvironmentName string
	Location        string
}

type Usage struct {
	g8sClient versioned.Interface
	k8sClient kubernetes.Interface
	logger    micrologger.Logger

	environmentName string
	location        string
}

func NewUsage(config UsageConfig) (*Usage, error) {
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
	if config.Location == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.Location must not be empty", config)
	}

	u := &Usage{
		g8sClient: config.G8sClient,
		k8sClient: config.K8sClient,
		logger:    config.Logger,

		environmentName: config.EnvironmentName,
		location:        config.Location,
	}

	return u, nil
}

func (u *Usage) Collect(ch chan<- prometheus.Metric) error {
	var crs []providerv1alpha1.AzureConfig
	{
		mark := ""
		page := 0
		for page == 0 || len(mark) > 0 {
			opts := metav1.ListOptions{
				Continue: mark,
			}
			list, err := u.g8sClient.ProviderV1alpha1().AzureConfigs("").List(opts)
			if err != nil {
				return microerror.Mask(err)
			}

			crs = append(crs, list.Items...)

			mark = list.Continue
			page++
		}
	}

	for _, cr := range crs {
		usageClient, err := u.getUsageClient(cr)
		if err != nil {
			return microerror.Mask(err)
		}

		r, err := usageClient.List(context.Background(), u.location)
		if err != nil {
			return microerror.Mask(err)
		}

		for r.NotDone() {
			for _, v := range r.Values() {
				fmt.Printf("\n")
				fmt.Printf("%#v\n", u.location)
				fmt.Printf("    %#v\n", *v.CurrentValue)
				fmt.Printf("    %#v\n", *v.Limit)
				fmt.Printf("    %#v\n", *v.Name)
				fmt.Printf("    %#v\n", *v.Unit)
				fmt.Printf("\n")
				//ch <- prometheus.MustNewConstMetric(
				//	deploymentDesc,
				//	prometheus.GaugeValue,
				//	float64(matchedStringToInt(statusRunning, *v.Properties.ProvisioningState)),
				//	key.ClusterID(cr),
				//	*v.Name,
				//	statusRunning,
				//)
				//ch <- prometheus.MustNewConstMetric(
				//	deploymentDesc,
				//	prometheus.GaugeValue,
				//	float64(matchedStringToInt(statusSucceeded, *v.Properties.ProvisioningState)),
				//	key.ClusterID(cr),
				//	*v.Name,
				//	statusSucceeded,
				//)
			}

			err := r.Next()
			if err != nil {
				return microerror.Mask(err)
			}
		}
	}

	return nil
}

func (u *Usage) Describe(ch chan<- *prometheus.Desc) error {
	ch <- usageCurrentDesc
	ch <- usageLimitDesc
	return nil
}

func (u *Usage) getUsageClient(cr providerv1alpha1.AzureConfig) (*compute.UsageClient, error) {
	config, err := credential.GetAzureConfig(u.k8sClient, key.CredentialName(cr), key.CredentialNamespace(cr))
	if err != nil {
		return nil, microerror.Mask(err)
	}
	config.EnvironmentName = u.environmentName

	azureClients, err := client.NewAzureClientSet(*config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return azureClients.UsageClient, nil
}
