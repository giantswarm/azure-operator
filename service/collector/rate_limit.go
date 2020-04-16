package collector

import (
	"context"
	"fmt"
	"strconv"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/pkg/project"
	"github.com/giantswarm/azure-operator/service/credential"
)

const (
	remainingReadsHeaderName  = "x-ms-ratelimit-remaining-subscription-reads"
	remainingWritesHeaderName = "x-ms-ratelimit-remaining-subscription-writes"
	resourceGroupNamePrefix   = "azure-operator-empty-rg-for-metrics"
	metricsNamespace          = "azure_operator"
	metricsSubsystem          = "rate_limit"
)

var (
	readsDesc = prometheus.NewDesc(
		prometheus.BuildFQName(metricsNamespace, metricsSubsystem, "reads"),
		"Remaining number of reads allowed.",
		[]string{
			"subscription",
			"clientid",
		},
		nil,
	)
	writesDesc = prometheus.NewDesc(
		prometheus.BuildFQName(metricsNamespace, metricsSubsystem, "writes"),
		"Remaining number of writes allowed.",
		[]string{
			"subscription",
			"clientid",
		},
		nil,
	)
	readsErrorCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metricsNamespace,
		Subsystem: metricsSubsystem,
		Name:      "reads_parsing_errors",
		Help:      "Errors trying to parse the remaining requests from the response header",
	})
	writesErrorCounter = prometheus.NewCounter(prometheus.CounterOpts{
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
	clientSets, err := credential.GetAzureClientSetsFromCredentialSecrets(u.k8sClient, u.environmentName)
	if err != nil {
		return microerror.Mask(err)
	}

	// The operator potentially uses a different set of credentials than
	// tenant clusters, so we add the operator credentials as well.
	operatorClientSet, err := client.NewAzureClientSet(u.cpAzureClientSetConfig)
	if err != nil {
		return microerror.Mask(err)
	}
	clientSets[&u.cpAzureClientSetConfig] = operatorClientSet

	ctx := context.Background()

	// We track RateLimit metrics for each client labeled by SubscriptionID and
	// ClientID.
	// That way we prevent duplicated metrics.
	for clientConfig, clientSet := range clientSets {
		// Remaining write requests can be retrieved sending a write request.
		var writes float64
		{
			resourceGroup := resources.Group{
				ManagedBy: to.StringPtr(project.Name()),
				Location:  to.StringPtr(u.location),
				Tags: map[string]*string{
					"collector": to.StringPtr(project.Name()),
				},
			}
			resourceGroup, err := clientSet.GroupsClient.CreateOrUpdate(ctx, u.getResourceGroupName(), resourceGroup)
			if err != nil {
				return microerror.Mask(err)
			}

			writes, err = strconv.ParseFloat(resourceGroup.Response.Header.Get(remainingWritesHeaderName), 64)
			if err != nil {
				u.logger.Log("level", "warning", "message", "an error occurred parsing to float the value inside the rate limiting header for write requests", "stack", microerror.JSON(microerror.Mask(err))) // nolint: errcheck
				writes = 0
				writesErrorCounter.Inc()
			}

			ch <- prometheus.MustNewConstMetric(
				writesDesc,
				prometheus.GaugeValue,
				writes,
				clientSet.GroupsClient.SubscriptionID,
				clientConfig.ClientID,
			)
		}

		// Remaining read requests can be retrieved sending a read request.
		var reads float64
		{
			groupResponse, err := clientSet.GroupsClient.Get(ctx, u.getResourceGroupName())
			if err != nil {
				return microerror.Mask(err)
			}

			reads, err = strconv.ParseFloat(groupResponse.Response.Header.Get(remainingReadsHeaderName), 64)
			if err != nil {
				u.logger.Log("level", "warning", "message", "an error occurred parsing to float the value inside the rate limiting header for read requests", "stack", microerror.JSON(microerror.Mask(err))) // nolint: errcheck
				reads = 0
				readsErrorCounter.Inc()
			}

			ch <- prometheus.MustNewConstMetric(
				readsDesc,
				prometheus.GaugeValue,
				reads,
				clientSet.GroupsClient.SubscriptionID,
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

func (u *RateLimit) getResourceGroupName() string {
	return fmt.Sprintf("%s-%s", resourceGroupNamePrefix, u.location)
}
