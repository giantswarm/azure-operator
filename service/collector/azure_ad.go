package collector

import (
	"context"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/controller/setting"
)

var (
	azureADDesc *prometheus.Desc = prometheus.NewDesc(
		prometheus.BuildFQName("azure_operator", "credentials", "expiration_timestamp"),
		"Azure credentials informations.",
		[]string{
			"application_id",
			"key_id",
		},
		nil,
	)
)

type AzureADConfig struct {
	K8sClient kubernetes.Interface
	Logger    micrologger.Logger

	AzureSetting setting.Azure
	// EnvironmentName is the name of the Azure environment used to compute the
	// azure.Environment type. See also
	// https://godoc.org/github.com/Azure/go-autorest/autorest/azure#Environment.
	EnvironmentName          string
	HostAzureClientSetConfig client.AzureClientSetConfig
}

type AzureAD struct {
	k8sClient kubernetes.Interface
	logger    micrologger.Logger

	environmentName          string
	hostAzureClientSetConfig client.AzureClientSetConfig
}

func NewAzureAD(config AzureADConfig) (*AzureAD, error) {
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if config.EnvironmentName == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.EnvironmentName must not be empty", config)
	}
	if err := config.HostAzureClientSetConfig.Validate(); err != nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.HostAzureClientSetConfig.%s", config, err)
	}

	a := &AzureAD{
		k8sClient: config.K8sClient,
		logger:    config.Logger,

		environmentName:          config.EnvironmentName,
		hostAzureClientSetConfig: config.HostAzureClientSetConfig,
	}

	return a, nil
}

func (a *AzureAD) Collect(ch chan<- prometheus.Metric) error {
	ctx := context.Background()

	// TODO call for CP client set
	clientSetConfigs, err := getAzureClientSetConfigs(a.k8sClient, a.environmentName)
	if err != nil {
		return microerror.Mask(err)
	}

	for _, c := range clientSetConfigs {
		err := a.collectForClientSetConfig(ctx, ch, c)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

func (a *AzureAD) Describe(ch chan<- *prometheus.Desc) error {
	ch <- resourceGroupDesc

	return nil
}

func (a *AzureAD) collectForClientSetConfig(ctx context.Context, ch chan<- prometheus.Metric, clientSetConfig *client.AzureClientSetConfig) error {
	clientSet, err := client.NewAzureClientSet(*clientSetConfig)
	if err != nil {
		return microerror.Mask(err)
	}

	// TODO handle non-authorized.
	result, err := clientSet.ApplicationsClient.ListPasswordCredentials(ctx, clientSetConfig.ClientID)
	if err != nil {
		return microerror.Mask(err)
	}

	if result.Value == nil {
		return nil
	}

	for _, credential := range *result.Value {
		var keyID string
		if credential.KeyID != nil {
			keyID = *credential.KeyID
		}

		ch <- prometheus.MustNewConstMetric(
			azureADDesc,
			prometheus.GaugeValue,
			float64(credential.EndDate.Unix()),
			clientSetConfig.ClientID,
			keyID,
		)
	}

	return nil
}
