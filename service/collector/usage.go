package collector

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/prometheus/client_golang/prometheus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/controller/key"
	"github.com/giantswarm/azure-operator/service/credential"
)

var (
	usageCurrentDesc *prometheus.Desc = prometheus.NewDesc(
		prometheus.BuildFQName("azure_operator", "usage", "current"),
		"Current usage of specific Quotas as defined by Azure.",
		[]string{
			"name",
			"subscription",
		},
		nil,
	)
	usageLimitDesc *prometheus.Desc = prometheus.NewDesc(
		prometheus.BuildFQName("azure_operator", "usage", "limit"),
		"Usage limit of specific Quotas as defined by Azure.",
		[]string{
			"name",
			"subscription",
		},
		nil,
	)
	scrapeErrorCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: component,
		Name:      "scrape_error",
		Help:      "Total number of times compute resource usage information scraping returned an error.",
	})
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

	usageScrapeError prometheus.Counter

	environmentName string
	location        string
}

const namespace = "azure_operator"
const component = "usage"

func init() {
	prometheus.MustRegister(scrapeErrorCounter)
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

		usageScrapeError: scrapeErrorCounter,

		environmentName: config.EnvironmentName,
		location:        config.Location,
	}

	return u, nil
}

func (u *Usage) Collect(ch chan<- prometheus.Metric) error {
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
	clients := map[string]*compute.UsageClient{}
	{
		for _, cr := range crs {
			c, err := u.getUsageClient(cr)
			if err != nil {
				return microerror.Mask(err)
			}

			clients[c.SubscriptionID] = c
		}
	}

	// We track usage metrics for each client labeled by subscription.
	// That way we prevent duplicated metrics.
	for subscriptionID, c := range clients {
		r, err := c.List(context.Background(), u.location)
		if err != nil {
			u.logger.Log("level", "warning", "message", "an error occurred during the scraping of current compute resource usage information", "stack", fmt.Sprintf("%v", err))
			u.usageScrapeError.Inc()
		} else {
			for r.NotDone() {
				for _, v := range r.Values() {
					ch <- prometheus.MustNewConstMetric(
						usageCurrentDesc,
						prometheus.GaugeValue,
						float64(*v.CurrentValue),
						*v.Name.LocalizedValue,
						subscriptionID,
					)
					ch <- prometheus.MustNewConstMetric(
						usageLimitDesc,
						prometheus.GaugeValue,
						float64(*v.Limit),
						*v.Name.LocalizedValue,
						subscriptionID,
					)
				}

				err := r.Next()
				if err != nil {
					return microerror.Mask(err)
				}
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
