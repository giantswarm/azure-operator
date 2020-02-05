package collector

import (
	"context"
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
	"github.com/giantswarm/azure-operator/service/controller/v12/key"
	"github.com/giantswarm/azure-operator/service/credential"
)

const (
	remainingReadsHeaderName  = "x-ms-ratelimit-remaining-subscription-reads"
	remainingWritesHeaderName = "x-ms-ratelimit-remaining-subscription-writes"
	resourceGroupName         = "azure-operator-empty-rg-for-metrics"
	metricsNamespace          = "azure_operator"
	metricsSubsystem          = "rate_limit"
)

var (
	readsDesc *prometheus.Desc = prometheus.NewDesc(
		prometheus.BuildFQName(metricsNamespace, metricsSubsystem, "reads"),
		"Remaining number of reads allowed.",
		[]string{
			"subscription",
			"clientid",
		},
		nil,
	)
	writesDesc *prometheus.Desc = prometheus.NewDesc(
		prometheus.BuildFQName(metricsNamespace, metricsSubsystem, "writes"),
		"Remaining number of writes allowed.",
		[]string{
			"subscription",
			"clientid",
		},
		nil,
	)
	readsErrorCounter prometheus.Counter = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metricsNamespace,
		Subsystem: metricsSubsystem,
		Name:      "reads_parsing_errors",
		Help:      "Errors trying to parse the remaining requests from the response header",
	})
	writesErrorCounter prometheus.Counter = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metricsNamespace,
		Subsystem: metricsSubsystem,
		Name:      "writes_parsing_errors",
		Help:      "Errors trying to parse the remaining requests from the response header",
	})
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

func init() {
	prometheus.MustRegister(readsErrorCounter)
	prometheus.MustRegister(writesErrorCounter)
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

		// The operator potentially uses a different set of credentials than
		// tenant clusters, so we add the operator credentials as well.
		config := &client.AzureClientSetConfig{
			ClientID:       u.cpAzureClientSetConfig.ClientID,
			ClientSecret:   u.cpAzureClientSetConfig.ClientSecret,
			SubscriptionID: u.cpAzureClientSetConfig.SubscriptionID,
			TenantID:       u.cpAzureClientSetConfig.TenantID,
		}
		clientConfigBySubscription[config.SubscriptionID] = *config
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

		// Remaining write requests can be retrieved sending a write request.
		var writes float64
		{
			resourceGroup := resources.Group{
				ManagedBy: to.StringPtr("azure-operator"),
				Location:  to.StringPtr(u.location),
				Tags: map[string]*string{
					"collector": to.StringPtr("azure-operator"),
				},
			}
			resourceGroup, err := azureClients.GroupsClient.CreateOrUpdate(ctx, resourceGroupName, resourceGroup)
			if err != nil {
				return microerror.Mask(err)
			}

			writes, err = strconv.ParseFloat(resourceGroup.Response.Header.Get(remainingWritesHeaderName), 64)
			if err != nil {
				u.logger.Log("level", "warning", "message", "an error occurred parsing to float the value inside the rate limiting header for write requests", "stack", microerror.Stack(microerror.Mask(err)))
				writes = 0
				writesErrorCounter.Inc()
				ch <- writesErrorCounter
			}

			ch <- prometheus.MustNewConstMetric(
				writesDesc,
				prometheus.GaugeValue,
				writes,
				azureClients.GroupsClient.SubscriptionID,
				clientConfig.ClientID,
			)
		}

		// Remaining read requests can be retrieved sending a read request.
		var reads float64
		{
			groupResponse, err := azureClients.GroupsClient.Get(ctx, resourceGroupName)
			if err != nil {
				return microerror.Mask(err)
			}

			reads, err = strconv.ParseFloat(groupResponse.Response.Header.Get(remainingReadsHeaderName), 64)
			if err != nil {
				u.logger.Log("level", "warning", "message", "an error occurred parsing to float the value inside the rate limiting header for read requests", "stack", microerror.Stack(microerror.Mask(err)))
				reads = 0
				readsErrorCounter.Inc()
				ch <- readsErrorCounter
			}

			ch <- prometheus.MustNewConstMetric(
				readsDesc,
				prometheus.GaugeValue,
				reads,
				azureClients.GroupsClient.SubscriptionID,
				clientConfig.ClientID,
			)
		}
	}

	return nil
}

func (u *RateLimit) Describe(ch chan<- *prometheus.Desc) error {
	ch <- readsDesc
	ch <- writesDesc
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
