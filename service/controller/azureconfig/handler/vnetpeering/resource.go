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
	TCAzureClientSet       *client.AzureClientSet
	CPAzureClientSet       *client.AzureClientSet
	HostResourceGroup      string
	HostVirtualNetworkName string
	Logger                 micrologger.Logger
}

type Resource struct {
	tcAzureClientSet       *client.AzureClientSet
	cpAzureClientSet       *client.AzureClientSet
	hostResourceGroup      string
	hostVirtualNetworkName string
	logger                 micrologger.Logger
}

func New(config Config) (*Resource, error) {
	if config.TCAzureClientSet == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.TCAzureClientSet must not be empty", config)
	}

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
		tcAzureClientSet:       config.TCAzureClientSet,
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
