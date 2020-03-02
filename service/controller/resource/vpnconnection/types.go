package vpnconnection

import (
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-11-01/network"
)

type connections struct {
	Guest network.VirtualNetworkGatewayConnection
	Host  network.VirtualNetworkGatewayConnection
}

func (c connections) isEmpty() bool {
	return c.Host.VirtualNetworkGatewayConnectionPropertiesFormat == nil || c.Guest.VirtualNetworkGatewayConnectionPropertiesFormat == nil
}
