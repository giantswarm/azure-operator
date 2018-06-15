package vpngateway

import (
	"github.com/giantswarm/microerror"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2017-09-01/network"
)

// toVnePeering convert v to network.VirtualNetworkPeering.
// If v is nil and empty network.VirtualNetworkPeering is returned.
func toVnetPeering(v interface{}) (network.VirtualNetworkPeering, error) {
	if v == nil {
		return network.VirtualNetworkPeering{}, nil
	}

	vnetPeering, ok := v.(network.VirtualNetworkPeering)
	if !ok {
		return network.VirtualNetworkPeering{}, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", network.VirtualNetworkPeering{}, v)
	}

	return vnetPeering, nil
}

// isVNetPeeringEmpty check whether peering correspond to network.VirtualNetworkPeering zero value.
func isVNetPeeringEmpty(peering network.VirtualNetworkPeering) bool {
	return peering == network.VirtualNetworkPeering{}
}
