package collector

import (
	"context"
	"fmt"
	"strconv"
	"strings"

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

const (
	vmssVMListHeaderName = "X-Ms-Ratelimit-Remaining-Resource"
)

var (
	vmssVMListDesc *prometheus.Desc = prometheus.NewDesc(
		prometheus.BuildFQName(metricsNamespace, metricsSubsystem, "vmss_instance_list"),
		"Remaining number of VMSS VM list operations.",
		[]string{
			"subscription",
			"clientid",
			"countername",
		},
		nil,
	)
	vmssVMListErrorCounter prometheus.Counter = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metricsNamespace,
		Subsystem: metricsSubsystem,
		Name:      "vmss_instance_list_parsing_errors",
		Help:      "Errors trying to parse the remaining requests from the response header",
	})
)

type VMSSRateLimitConfig struct {
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

type VMSSRateLimit struct {
	g8sClient versioned.Interface
	k8sClient kubernetes.Interface
	logger    micrologger.Logger

	environmentName        string
	location               string
	cpAzureClientSetConfig client.AzureClientSetConfig
}

func init() {
	prometheus.MustRegister(vmssVMListErrorCounter)
}

func NewVMSSRateLimit(config VMSSRateLimitConfig) (*VMSSRateLimit, error) {
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

	u := &VMSSRateLimit{
		g8sClient: config.G8sClient,
		k8sClient: config.K8sClient,
		logger:    config.Logger,

		environmentName:        config.EnvironmentName,
		location:               config.Location,
		cpAzureClientSetConfig: config.CPAzureClientSetConfig,
	}

	return u, nil
}

func (u *VMSSRateLimit) Collect(ch chan<- prometheus.Metric) error {
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

	ctx := context.Background()

	{
		var doneSubscriptions []string
		for _, cr := range crs {
			config, err := credential.GetAzureConfig(u.k8sClient, key.CredentialName(cr), key.CredentialNamespace(cr))
			if err != nil {
				return microerror.Mask(err)
			}

			// We want to check only once per subscriptino
			if inArray(doneSubscriptions, config.SubscriptionID) {
				continue
			}
			azureClients, err := client.NewAzureClientSet(*config)
			if err != nil {
				return microerror.Mask(err)
			}

			// VMSS List VMs specific limits.
			{
				// We need to find a resource group for an existing tenant cluster in this subscription.
				result, err := azureClients.VirtualMachineScaleSetVMsClient.ListComplete(ctx, cr.Name, fmt.Sprintf("%s-worker", cr.Name), "", "", "")
				if err != nil {
					u.logger.LogCtx(ctx, fmt.Sprintf("Error listing VM instances on %s: %s", cr.Name, err.Error()))
					continue
				}

				limits := result.Response().Response.Header[vmssVMListHeaderName]

				// Header not found, we consider this an error.
				if len(limits) == 0 {
					vmssVMListErrorCounter.Inc()
					ch <- vmssVMListErrorCounter
					continue
				}

				for _, l := range limits {
					// Limits are a single comma separated string.
					tokens := strings.SplitN(l, ",", -1)
					for _, t := range tokens {
						// Each limit's name and value are separated by a semicolon.
						kv := strings.SplitN(t, ";", 2)
						if len(kv) != 2 {
							// We expect exactly two tokens, otherwise we consider this a parsing error.
							vmssVMListErrorCounter.Inc()
							ch <- vmssVMListErrorCounter
							continue
						}

						// The second token must be a number or we don't know what we got from MS.
						val, err := strconv.ParseFloat(kv[1], 64)
						if err != nil {
							vmssVMListErrorCounter.Inc()
							ch <- vmssVMListErrorCounter
							continue
						}

						ch <- prometheus.MustNewConstMetric(
							vmssVMListDesc,
							prometheus.GaugeValue,
							val,
							config.SubscriptionID,
							config.ClientID,
							kv[0],
						)
					}
				}
			}
		}
	}

	return nil
}

func (u *VMSSRateLimit) Describe(ch chan<- *prometheus.Desc) error {
	ch <- vmssVMListDesc
	return nil
}

func (u *VMSSRateLimit) getAzureClients(cr providerv1alpha1.AzureConfig) (*client.AzureClientSetConfig, *client.AzureClientSet, error) {
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

func inArray(a []string, s string) bool {
	for _, x := range a {
		if x == s {
			return true
		}
	}

	return false
}
