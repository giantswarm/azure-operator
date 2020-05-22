package vnetpeering

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-11-01/network"
	"github.com/giantswarm/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/azure-operator/v4/service/controller/controllercontext"
)

const (
	Name = "vnetpeering"
)

type Config struct {
	CPVnetPeeringsClient   *network.VirtualNetworkPeeringsClient
	HostResourceGroup      string
	HostVirtualNetworkName string
	K8sClient              k8sclient.Interface
	Logger                 micrologger.Logger
}

type Resource struct {
	cpVnetPeeringsClient   *network.VirtualNetworkPeeringsClient
	hostResourceGroup      string
	hostVirtualNetworkName string
	k8sClient              k8sclient.Interface
	logger                 micrologger.Logger
}

func New(config Config) (*Resource, error) {
	if config.HostResourceGroup == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.HostResourceGroup must not be empty", config)
	}

	if config.HostVirtualNetworkName == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.HostVirtualNetworkName must not be empty", config)
	}

	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}

	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	r := &Resource{
		cpVnetPeeringsClient:   config.CPVnetPeeringsClient,
		hostResourceGroup:      config.HostResourceGroup,
		hostVirtualNetworkName: config.HostVirtualNetworkName,
		k8sClient:              config.K8sClient,
		logger:                 config.Logger,
	}

	return r, nil
}

func (r *Resource) Name() string {
	return Name
}

func (r *Resource) getPublicIPAddressesClient(ctx context.Context) (*network.PublicIPAddressesClient, error) {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return cc.AzureClientSet.PublicIPAddressesClient, nil
}

func (r *Resource) getTCVnetPeeringsClient(ctx context.Context) (*network.VirtualNetworkPeeringsClient, error) {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return cc.AzureClientSet.VnetPeeringClient, nil
}

func (r *Resource) getTCVnetClient(ctx context.Context) (*network.VirtualNetworksClient, error) {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return cc.AzureClientSet.VirtualNetworkClient, nil
}

func (r *Resource) getVnetGatewaysClient(ctx context.Context) (*network.VirtualNetworkGatewaysClient, error) {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return cc.AzureClientSet.VirtualNetworkGatewaysClient, nil
}

func (r *Resource) getVnetGatewaysConnectionsClient(ctx context.Context) (*network.VirtualNetworkGatewayConnectionsClient, error) {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return cc.AzureClientSet.VirtualNetworkGatewayConnectionsClient, nil
}
