package ipam

import (
	"context"
	"net"

	"github.com/giantswarm/microerror"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

var nodePoolIPMask = net.CIDRMask(24, 32)

type AzureMachinePoolNetworkRangeGetterConfig struct {
	Client client.Client
}

// AzureMachinePoolNetworkRangeGetter is a NetworkRangeGetter and a
// NetworkRangeScopeGetter implementation for node pools.
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

	// Reads cluster.x-k8s.io/cluster-name label from AzureMachinePool CR and then gets Cluster CR
	// by that name.
	cluster, err := util.GetClusterFromMetadata(ctx, g.client, azureMachinePool.ObjectMeta)
	if err != nil {
		return net.IPNet{}, microerror.Mask(err)
	}

	if cluster.Spec.ClusterNetwork == nil {
		err = microerror.Maskf(invalidObjectError, "%T.ClusterNetwork must not be empty", cluster.Spec)
		return net.IPNet{}, err
	}

	if cluster.Spec.ClusterNetwork.Services == nil {
		err = microerror.Maskf(invalidObjectError, "%T.Services must not be empty", cluster.Spec.ClusterNetwork)
		return net.IPNet{}, microerror.Mask(err)
	}

	if len(cluster.Spec.ClusterNetwork.Services.CIDRBlocks) == 0 {
		err = microerror.Maskf(invalidObjectError, "%T.CIDRBlocks must not be empty", cluster.Spec.ClusterNetwork.Services)
		return net.IPNet{}, err
	}

	cidrBlock := cluster.Spec.ClusterNetwork.Services.CIDRBlocks[0] // or should we use AzureCluster.Spec.NetworkSpec.Vnet.CidrBlock here?
	_, ipNet, err := net.ParseCIDR(cidrBlock)
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
