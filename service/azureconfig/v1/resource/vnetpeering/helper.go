package vnetpeering

import (
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2017-09-01/network"
)

func isVNetPeeringEmpty(peering network.VirtualNetworkPeering) bool {
	return peering == network.VirtualNetworkPeering{}
}
