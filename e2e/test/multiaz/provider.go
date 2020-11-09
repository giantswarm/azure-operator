// +build k8srequired

package multiaz

import (
	"context"
	"fmt"

	"github.com/giantswarm/apiextensions/v3/pkg/clientset/versioned"
	azureclient "github.com/giantswarm/e2eclients/azure"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
)

type ProviderConfig struct {
	AzureClient *azureclient.Client
	G8sClient   versioned.Interface
	Logger      micrologger.Logger

	ClusterID  string
	NodePoolID string
}

type Provider struct {
	azureClient *azureclient.Client
	g8sClient   versioned.Interface
	logger      micrologger.Logger

	clusterID  string
	nodePoolID string
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

	if config.NodePoolID == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.NodePoolID must not be empty", config)
	}

	p := &Provider{
		azureClient: config.AzureClient,
		g8sClient:   config.G8sClient,
		logger:      config.Logger,

		clusterID:  config.ClusterID,
		nodePoolID: config.NodePoolID,
	}

	return p, nil
}

func (p *Provider) GetClusterAZs(ctx context.Context) ([]string, error) {
	vmss, err := p.azureClient.VirtualMachineScaleSetsClient.Get(ctx, p.clusterID, fmt.Sprintf("nodepool-%s", p.nodePoolID))
	if err != nil {
		return []string{}, microerror.Mask(err)
	}

	return *vmss.Zones, nil
}
