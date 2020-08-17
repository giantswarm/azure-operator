package ipam

import (
	"context"
	"net"

	"github.com/giantswarm/microerror"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v4/pkg/helpers"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

var nodePoolIPMask = net.CIDRMask(24, 32)

type AzureMachinePoolNetworkRangeGetterConfig struct {
	Client client.Client
}

// AzureMachinePoolNetworkRangeGetter is a NetworkRangeGetter implementation for node pools.
type AzureMachinePoolNetworkRangeGetter struct {
	client client.Client
}

func NewAzureMachinePoolNetworkRangeGetter(config AzureMachinePoolNetworkRangeGetterConfig) (*AzureMachinePoolNetworkRangeGetter, error) {
	if config.Client == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Client must not be empty", config)
	}

	g := &AzureMachinePoolNetworkRangeGetter{
		client: config.Client,
	}

	return g, nil
}

// GetNetworkRange return the tenant cluster virtual network range, because the
// node pool subnet is getting its IP address range from all available address
// ranges in the tenant cluster virtual network.
func (g *AzureMachinePoolNetworkRangeGetter) GetNetworkRange(ctx context.Context, obj interface{}) (net.IPNet, error) {
	azureMachinePool, err := key.ToAzureMachinePool(obj)
	if err != nil {
		return net.IPNet{}, microerror.Mask(err)
	}

	// Get AzureCluster CR where the NetworkSpec is stored.
	azureCluster, err := helpers.GetAzureClusterFromMetadata(ctx, g.client, azureMachinePool.ObjectMeta)
	if err != nil {
		return net.IPNet{}, microerror.Mask(err)
	}

	_, ipNet, err := net.ParseCIDR(azureCluster.Spec.NetworkSpec.Vnet.CidrBlock)
	if err != nil {
		return net.IPNet{}, err
	}

	return *ipNet, nil
}

// GetRequiredMask returns an /24 IP mask that is required for the node pools
// subnet.
func (g *AzureMachinePoolNetworkRangeGetter) GetRequiredIPMask() net.IPMask {
	return nodePoolIPMask
}
