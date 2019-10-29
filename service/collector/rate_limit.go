package collector

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"

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
	"github.com/giantswarm/azure-operator/service/controller/v9/key"
	"github.com/giantswarm/azure-operator/service/credential"
)

const (
	remainingReadsHeaderName  = "x-ms-ratelimit-remaining-subscription-reads"
	remainingWritesHeaderName = "x-ms-ratelimit-remaining-subscription-writes"
)

var (
	ReadsDesc *prometheus.Desc = prometheus.NewDesc(
		prometheus.BuildFQName("azure_operator", "rate_limit", "reads"),
		"Remaining number of reads allowed.",
		[]string{
			"subscription",
			"clientid",
		},
		nil,
	)
	WritesDesc *prometheus.Desc = prometheus.NewDesc(
		prometheus.BuildFQName("azure_operator", "rate_limit", "writes"),
		"Remaining number of writes allowed.",
		[]string{
			"subscription",
			"clientid",
		},
		nil,
	)
)

type RateLimitConfig struct {
	G8sClient versioned.Interface
	K8sClient kubernetes.Interface
	Logger    micrologger.Logger

	// EnvironmentName is the name of the Azure environment used to compute the
	// azure.Environment type. See also
	// https://godoc.org/github.com/Azure/go-autorest/autorest/azure#Environment.
	EnvironmentName        string
	Location               string
	CPAzureClientSetConfig client.AzureClientSetConfig
}

type RateLimit struct {
	g8sClient versioned.Interface
	k8sClient kubernetes.Interface
	logger    micrologger.Logger

	environmentName        string
	location               string
	cpAzureClientSetConfig client.AzureClientSetConfig
}

func NewRateLimit(config RateLimitConfig) (*RateLimit, error) {
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

	u := &RateLimit{
		g8sClient: config.G8sClient,
		k8sClient: config.K8sClient,
		logger:    config.Logger,

		environmentName:        config.EnvironmentName,
		location:               config.Location,
		cpAzureClientSetConfig: config.CPAzureClientSetConfig,
	}

	return u, nil
}

func (u *RateLimit) Collect(ch chan<- prometheus.Metric) error {
	// We need all CRs to gather all subscriptions below.
	var crs []providerv1alpha1.AzureConfig
	{
		mark := ""
		page := 0
		for page == 0 || len(mark) > 0 {
			opts := metav1.ListOptions{
				Continue: mark,
			}
			list, err := u.g8sClient.ProviderV1alpha1().AzureConfigs(metav1.NamespaceAll).List(opts)
			if err != nil {
				return microerror.Mask(err)
			}

			crs = append(crs, list.Items...)

			mark = list.Continue
			page++
		}
	}

	// We generate clients and group them by subscription.
	// Subscription is defined by the Secret used to fetch the credentials.
	clientConfigBySubscription := map[string]client.AzureClientSetConfig{}
	{
		for _, cr := range crs {
			config, err := credential.GetAzureConfig(u.k8sClient, key.CredentialName(cr), key.CredentialNamespace(cr))
			if err != nil {
				return microerror.Mask(err)
			}
			clientConfigBySubscription[config.SubscriptionID] = *config
		}

		if _, present := clientConfigBySubscription[u.cpAzureClientSetConfig.SubscriptionID]; !present {
			config := &client.AzureClientSetConfig{
				ClientID:       u.cpAzureClientSetConfig.ClientID,
				ClientSecret:   u.cpAzureClientSetConfig.ClientSecret,
				SubscriptionID: u.cpAzureClientSetConfig.SubscriptionID,
				TenantID:       u.cpAzureClientSetConfig.TenantID,
			}
			clientConfigBySubscription[config.SubscriptionID] = *config
		}
	}

	ctx := context.Background()

	// We track RateLimit metrics for each client labeled by SubscriptionID and
	// ClientID.
	// That way we prevent duplicated metrics.
	for _, clientConfig := range clientConfigBySubscription {
		azureClients, err := client.NewAzureClientSet(clientConfig)
		if err != nil {
			return microerror.Mask(err)
		}

		resourceGroup, err := u.createEmptyResourceGroup(ctx, azureClients)
		if err != nil {
			return microerror.Mask(err)
		}

		// Remaining read requests can be retrieved sending a read request.
		var reads float64
		{
			groupResponse, err := azureClients.GroupsClient.Get(ctx, *resourceGroup.Name)
			if err != nil {
				return microerror.Mask(err)
			}

			reads, err = strconv.ParseFloat(groupResponse.Response.Header.Get(remainingReadsHeaderName), 64)
			if err != nil {
				u.logger.Log("level", "warning", "message", "an error occurred parsing to float the value inside the rate limiting header for read requests", "stack", microerror.Stack(microerror.Mask(err)))
				reads = 0
			}

			ch <- prometheus.MustNewConstMetric(
				ReadsDesc,
				prometheus.GaugeValue,
				reads,
				azureClients.GroupsClient.SubscriptionID,
				clientConfig.ClientID,
			)
		}

		// Remaining write requests can be retrieved sending a write request.
		var writes float64
		{
			deleteResponse, err := azureClients.GroupsClient.Delete(ctx, *resourceGroup.Name)
			if err != nil {
				return microerror.Mask(err)
			}

			writes, err = strconv.ParseFloat(deleteResponse.Response().Header.Get(remainingWritesHeaderName), 64)
			if err != nil {
				u.logger.Log("level", "warning", "message", "an error occurred parsing to float the value inside the rate limiting header for write requests", "stack", microerror.Stack(microerror.Mask(err)))
				writes = 0
			}

			ch <- prometheus.MustNewConstMetric(
				WritesDesc,
				prometheus.GaugeValue,
				writes,
				azureClients.GroupsClient.SubscriptionID,
				clientConfig.ClientID,
			)
		}
	}

	return nil
}

func (u *RateLimit) Describe(ch chan<- *prometheus.Desc) error {
	ch <- ReadsDesc
	ch <- WritesDesc
	return nil
}

func (u *RateLimit) getAzureClients(cr providerv1alpha1.AzureConfig) (*client.AzureClientSetConfig, *client.AzureClientSet, error) {
	config, err := credential.GetAzureConfig(u.k8sClient, key.CredentialName(cr), key.CredentialNamespace(cr))
	if err != nil {
		return nil, nil, microerror.Mask(err)
	}
	config.EnvironmentName = u.environmentName

	azureClients, err := client.NewAzureClientSet(*config)
	if err != nil {
		return nil, nil, microerror.Mask(err)
	}

	return config, azureClients, nil
}

// We create a resource group with a random name so we can later send
// requests to the API and figure out the rate limiting status.
func (u *RateLimit) createEmptyResourceGroup(ctx context.Context, azureClients *client.AzureClientSet) (resources.Group, error) {
	resourceGroupName := randStringBytes(16)
	resourceGroup := resources.Group{
		ManagedBy: to.StringPtr("azure-operator"),
		Location:  to.StringPtr(u.location),
		Tags: map[string]*string{
			"collector": to.StringPtr("azure-operator"),
		},
	}
	return azureClients.GroupsClient.CreateOrUpdate(ctx, resourceGroupName, resourceGroup)
}

const (
	letterBytes             = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"
	resourceGroupNamePrefix = "azure-operator-collector"
)

func randStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}

	return fmt.Sprintf("%s-%s", resourceGroupNamePrefix, b)
}
