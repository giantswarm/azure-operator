// +build k8srequired

package multiaz

import (
	"context"

	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	azureclient "github.com/giantswarm/e2eclients/azure"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ProviderConfig struct {
	AzureClient *azureclient.Client
	G8sClient   versioned.Interface
	Logger      micrologger.Logger

	ClusterID string
}

type Provider struct {
	azureClient *azureclient.Client
	g8sClient   versioned.Interface
	logger      micrologger.Logger

	clusterID string
}

func NewProvider(config ProviderConfig) (*Provider, error) {
	if config.AzureClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.AzureClient must not be empty", config)
	}
	if config.G8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.G8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if config.ClusterID == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.ClusterID must not be empty", config)
	}

	p := &Provider{
		azureClient: config.AzureClient,
		g8sClient:   config.G8sClient,
		logger:      config.Logger,

		clusterID: config.ClusterID,
	}

	return p, nil
}

func (p *Provider) GetClusterAZs(ctx context.Context) ([]int, error) {
	customResource, err := p.g8sClient.ProviderV1alpha1().AzureConfigs("default").Get(p.clusterID, metav1.GetOptions{})
	if err != nil {
		return []int{}, microerror.Mask(err)
	}

	return customResource.Status.Provider.AvailabilityZones, nil
}
