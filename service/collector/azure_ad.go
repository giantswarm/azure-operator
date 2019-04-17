package collector

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/controller/setting"
)

var (
	AzureAdDesc *prometheus.Desc = prometheus.NewDesc(
		prometheus.BuildFQName("azure_operator", "credentials", "info"),
		"Azure credentials informations.",
		[]string{
			"id",
			"name",
			"service_principal",
			"expiration_timestamp",
		},
		nil,
	)
)

type AzureAdConfig struct {
	K8sClient kubernetes.Interface
	Logger    micrologger.Logger

	AzureSetting             setting.Azure
	HostAzureClientSetConfig client.AzureClientSetConfig
}

type AzureAd struct {
	k8sClient kubernetes.Interface
	logger    micrologger.Logger

	azureSetting             setting.Azure
	hostAzureClientSetConfig client.AzureClientSetConfig
}

func NewAzureAd(config AzureAdConfig) (*AzureAd, error) {
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if err := config.AzureSetting.Validate(); err != nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.AzureSetting.%s", config, err)
	}
	if err := config.HostAzureClientSetConfig.Validate(); err != nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.HostAzureClientSetConfig.%s", config, err)
	}

	a := &AzureAd{
		k8sClient: config.K8sClient,
		logger:    config.Logger,

		azureSetting:             config.AzureSetting,
		hostAzureClientSetConfig: config.HostAzureClientSetConfig,
	}

	return a, nil
}

func (r *AzureAd) Collect(ch chan<- prometheus.Metric) error {
	fmt.Print("ey")

	return nil
}

func (r *AzureAd) Describe(ch chan<- *prometheus.Desc) error {
	ch <- resourceGroupDesc

	return nil
}

func (a *AzureAd) getApplicationsClient() (*graphrbac.ApplicationsClient, error) {
	azureClients, err := client.NewAzureClientSet(a.hostAzureClientSetConfig)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return azureClients.ApplicationsClient, nil
}

func (a *AzureAd) getApplicationExpirationDate(ctx context.Context) (string, error) {
	appClient, err := a.getApplicationsClient()
	if err != nil {
		return "", microerror.Mask(err)
	}

	appPasswords, err := appClient.ListPasswordCredentials(ctx, a.hostAzureClientSetConfig.ClientID)
	if err != nil {
		return "", microerror.Mask(err)
	}


	return , nil
}
