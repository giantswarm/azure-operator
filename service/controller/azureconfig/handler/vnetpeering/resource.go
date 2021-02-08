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
	ClientFactory          client.OrganizationFactory
	CPAzureClientSet       *client.AzureClientSet
	HostResourceGroup      string
	HostVirtualNetworkName string
	Logger                 micrologger.Logger
}

type Resource struct {
	clientFactory          client.OrganizationFactory
	cpAzureClientSet       *client.AzureClientSet
	hostResourceGroup      string
	hostVirtualNetworkName string
	logger                 micrologger.Logger
}

func New(config Config) (*Resource, error) {
	if config.CPAzureClientSet == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CPAzureClientSet must not be empty", config)
	}

	if config.HostResourceGroup == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.HostResourceGroup must not be empty", config)
	}

	if config.HostVirtualNetworkName == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.HostVirtualNetworkName must not be empty", config)
	}

	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	r := &Resource{
		clientFactory:          config.ClientFactory,
		cpAzureClientSet:       config.CPAzureClientSet,
		hostResourceGroup:      config.HostResourceGroup,
		hostVirtualNetworkName: config.HostVirtualNetworkName,
		logger:                 config.Logger,
	}

	return r, nil
}

func (r *Resource) Name() string {
	return Name
}
