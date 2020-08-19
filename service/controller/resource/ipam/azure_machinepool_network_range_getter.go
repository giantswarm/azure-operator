package ipam

import (
	"context"
	"fmt"
	"net"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v4/pkg/helpers"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

var nodePoolIPMask = net.CIDRMask(24, 32)

type AzureMachinePoolNetworkRangeGetterConfig struct {
	CtrlClient client.Client
	Logger     micrologger.Logger
}

// AzureMachinePoolNetworkRangeGetter is a NetworkRangeGetter implementation for node pools.
type AzureMachinePoolNetworkRangeGetter struct {
	client client.Client
	logger micrologger.Logger
}

func NewAzureMachinePoolNetworkRangeGetter(config AzureMachinePoolNetworkRangeGetterConfig) (*AzureMachinePoolNetworkRangeGetter, error) {
	if config.CtrlClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CtrlClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	g := &AzureMachinePoolNetworkRangeGetter{
		client: config.CtrlClient,
		logger: config.Logger,
	}

	return g, nil
}

// GetNetworkRange return the tenant cluster virtual network range, because the
// node pool subnet is getting its IP address range from all available address
// ranges in the tenant cluster virtual network.
func (g *AzureMachinePoolNetworkRangeGetter) GetNetworkRange(ctx context.Context, obj interface{}) (net.IPNet, error) {
	g.logger.LogCtx(
		ctx,
		"level", "debug",
		"message", "getting tenant cluster's VNet range from which the node pool subnet will be allocated")

	azureMachinePool, err := key.ToAzureMachinePool(obj)
	if err != nil {
		return net.IPNet{}, microerror.Mask(err)
	}

	// Get AzureCluster CR where the NetworkSpec is stored.
	azureCluster, err := helpers.GetAzureClusterFromMetadata(ctx, g.client, azureMachinePool.ObjectMeta)
	if err != nil {
		return net.IPNet{}, microerror.Mask(err)
	}

	if azureCluster.Spec.NetworkSpec.Vnet.CidrBlock == "" {
		// This can happen when AzureCluster.Spec.NetworkSpec.Vnet.CidrBlock is still not set,
		// because VNet for the tenant cluster is still not allocated (e.g. when cluster is still
		// being created).
		errorMessage := "AzureCluster.Spec.NetworkSpec.Vnet.CidrBlock is not set yet"
		g.logger.LogCtx(ctx, "level", "warning", "message", errorMessage)
		return net.IPNet{}, microerror.Maskf(networkRangeStillNotKnown, errorMessage)
	}

	_, ipNet, err := net.ParseCIDR(azureCluster.Spec.NetworkSpec.Vnet.CidrBlock)
	if err != nil {
		return net.IPNet{}, microerror.Mask(err)
	}

	g.logger.LogCtx(
		ctx,
		"level", "debug",
		"message", fmt.Sprintf("got tenant cluster's VNet range %s from which the node pool subnet will be allocated", ipNet.String()))

	return *ipNet, nil
}

// GetRequiredMask returns a /24 IP mask that is required for the node pools
// subnet.
func (g *AzureMachinePoolNetworkRangeGetter) GetRequiredIPMask() net.IPMask {
	return nodePoolIPMask
}
