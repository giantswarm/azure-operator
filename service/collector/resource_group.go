package collector

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/errgroup"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/credential"
)

const (
	labelID        = "id"
	labelName      = "name"
	labelState     = "state"
	labelLocation  = "location"
	labelManagedBy = "managed_by"
)

var (
	resourceGroupDesc = prometheus.NewDesc(
		prometheus.BuildFQName("azure_operator", "resource_group", "info"),
		"Resource group information.",
		[]string{
			labelID,
			labelName,
			labelState,
			labelLocation,
			labelManagedBy,
		},
		nil,
	)

	gaugeValue float64 = 1
)

type ResourceGroupConfig struct {
	K8sClient kubernetes.Interface
	Logger    micrologger.Logger

	EnvironmentName        string
	CPAzureClientSetConfig client.AzureClientSetConfig
}

type ResourceGroup struct {
	k8sClient kubernetes.Interface
	logger    micrologger.Logger

	environmentName        string
	cpAzureClientSetConfig client.AzureClientSetConfig
}

func NewResourceGroup(config ResourceGroupConfig) (*ResourceGroup, error) {
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if config.EnvironmentName == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.EnvironmentName must not be empty", config)
	}

	r := &ResourceGroup{
		k8sClient: config.K8sClient,
		logger:    config.Logger,

		environmentName:        config.EnvironmentName,
		cpAzureClientSetConfig: config.CPAzureClientSetConfig,
	}

	return r, nil
}

func (r *ResourceGroup) Collect(ch chan<- prometheus.Metric) error {
	ctx := context.Background()
	clientSets, err := credential.GetAzureClientSetsFromCredentialSecretsBySubscription(r.k8sClient, r.environmentName)
	if err != nil {
		return microerror.Mask(err)
	}

	// The operator potentially uses a different set of credentials than
	// tenant clusters, so we add the operator credentials as well.
	operatorClientSet, err := client.NewAzureClientSet(r.cpAzureClientSetConfig)
	if err != nil {
		return microerror.Mask(err)
	}
	clientSets[r.cpAzureClientSetConfig.SubscriptionID] = operatorClientSet

	var g errgroup.Group

	for _, item := range clientSets {
		clientSet := item

		g.Go(func() error {
			err := r.collectForClientSet(ctx, ch, clientSet.GroupsClient)
			if err != nil {
				return microerror.Mask(err)
			}

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *ResourceGroup) collectForClientSet(ctx context.Context, ch chan<- prometheus.Metric, client *resources.GroupsClient) error {
	resultsPage, err := client.ListComplete(context.Background(), "", nil)
	if err != nil {
		return microerror.Mask(err)
	}

	for resultsPage.NotDone() {
		group := resultsPage.Value()
		ch <- prometheus.MustNewConstMetric(
			resourceGroupDesc,
			prometheus.GaugeValue,
			gaugeValue,
			to.String(group.ID),
			to.String(group.Name),
			getState(group),
			to.String(group.Location),
			to.String(group.ManagedBy),
		)

		if err := resultsPage.NextWithContext(ctx); err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

func (r *ResourceGroup) Describe(ch chan<- *prometheus.Desc) error {
	ch <- resourceGroupDesc

	return nil
}

func getState(group resources.Group) string {
	if group.Properties != nil {
		return to.String(group.Properties.ProvisioningState)
	}

	return ""
}
