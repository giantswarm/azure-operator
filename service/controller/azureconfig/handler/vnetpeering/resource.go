package vnetpeering

import (
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/azure-operator/v5/client"
)

const (
	Name = "vnetpeering"
)

type Config struct {
	ClientFactory        client.OrganizationFactory
	CPAzureClientSet     *client.AzureClientSet
	MCResourceGroup      string
	MCVirtualNetworkName string
	Logger               micrologger.Logger
}

type Resource struct {
	clientFactory        client.OrganizationFactory
	cpAzureClientSet     *client.AzureClientSet
	mcResourceGroup      string
	mcVirtualNetworkName string
	logger               micrologger.Logger
}

func New(config Config) (*Resource, error) {
	if config.CPAzureClientSet == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CPAzureClientSet must not be empty", config)
	}

	if config.MCResourceGroup == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.MCResourceGroup must not be empty", config)
	}

	if config.MCVirtualNetworkName == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.MCVirtualNetworkName must not be empty", config)
	}

	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	r := &Resource{
		clientFactory:        config.ClientFactory,
		cpAzureClientSet:     config.CPAzureClientSet,
		mcResourceGroup:      config.MCResourceGroup,
		mcVirtualNetworkName: config.MCVirtualNetworkName,
		logger:               config.Logger,
	}

	return r, nil
}

func (r *Resource) Name() string {
	return Name
}
