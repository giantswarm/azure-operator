package collector

import (
	"context"
	"fmt"
	"strconv"

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
	VMSSReadsDesc *prometheus.Desc = prometheus.NewDesc(
		prometheus.BuildFQName("azure_operator", "VMSS", "reads"),
		"Remaining number of reads allowed.",
		[]string{
			"subscription",
			"clientid",
		},
		nil,
	)
)

type VMSSConfig struct {
	G8sClient versioned.Interface
	K8sClient kubernetes.Interface
	Logger    micrologger.Logger

	// EnvironmentName is the name of the Azure environment used to compute the
	// azure.Environment type. See also
	// https://godoc.org/github.com/Azure/go-autorest/autorest/azure#Environment.
	EnvironmentName string
}

type VMSS struct {
	g8sClient versioned.Interface
	k8sClient kubernetes.Interface
	logger    micrologger.Logger

	environmentName string
}

func NewVMSS(config VMSSConfig) (*VMSS, error) {
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

	u := &VMSS{
		g8sClient: config.G8sClient,
		k8sClient: config.K8sClient,
		logger:    config.Logger,

		environmentName: config.EnvironmentName,
	}

	return u, nil
}

func (u *VMSS) Collect(ch chan<- prometheus.Metric) error {
	// We need all CRs to gather all subscriptions below.
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

	// We generate clients and group them by subscription.
	// Subscription is defined by the credentials used.
	crsBySubscription := map[string]providerv1alpha1.AzureConfig{}
	{
		for _, cr := range crs {
			subscriptionCredentials := fmt.Sprintf("%s-%s", key.CredentialName(cr), key.CredentialNamespace(cr))
			crsBySubscription[subscriptionCredentials] = cr
		}
	}

	ctx := context.Background()

	// We track VMSS metrics for each client labeled by ClientID.
	// That way we prevent duplicated metrics.
	for _, cr := range crsBySubscription {
		azureConfig, vmssClient, err := u.getVMSSClient(cr)
		if err != nil {
			return microerror.Mask(err)
		}

		r, err := vmssClient.ListAll(ctx)
		if err != nil {
			u.logger.Log("level", "warning", "message", "an error occurred during the scraping of current compute resource VMSS information", "stack", fmt.Sprintf("%v", err))
		} else {
			reads, err := strconv.ParseInt(r.Response().Header.Get("x-ms-ratelimit-remaining-subscription-reads"), 10, 64)
			if err != nil {
				reads = 0
			}

			ch <- prometheus.MustNewConstMetric(
				VMSSReadsDesc,
				prometheus.GaugeValue,
				float64(reads),
				vmssClient.SubscriptionID,
				azureConfig.ClientID,
			)
		}
	}

	return nil
}

func (u *VMSS) Describe(ch chan<- *prometheus.Desc) error {
	ch <- VMSSReadsDesc
	return nil
}

func (u *VMSS) getVMSSClient(cr providerv1alpha1.AzureConfig) (*client.AzureClientSetConfig, *compute.VirtualMachineScaleSetsClient, error) {
	config, err := credential.GetAzureConfig(u.k8sClient, key.CredentialName(cr), key.CredentialNamespace(cr))
	if err != nil {
		return nil, nil, microerror.Mask(err)
	}
	config.EnvironmentName = u.environmentName

	azureClients, err := client.NewAzureClientSet(*config)
	if err != nil {
		return config, nil, microerror.Mask(err)
	}

	return config, azureClients.VirtualMachineScaleSetsClient, nil
}
