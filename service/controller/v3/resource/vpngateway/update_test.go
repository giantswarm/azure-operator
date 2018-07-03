package vpngateway

import (
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2017-09-01/network"
	"github.com/Azure/go-autorest/autorest/to"
)

func TestNeedUpdate(t *testing.T) {
	testCases := []struct {
		description      string
		current, desired network.VirtualNetworkGatewayConnection
		expected         bool
	}{
		{
			"desired state empty",
			network.VirtualNetworkGatewayConnection{},
			network.VirtualNetworkGatewayConnection{},
			false,
		},
		{
			"name",
			network.VirtualNetworkGatewayConnection{
				Name: to.StringPtr("wrong name"),
			},
			network.VirtualNetworkGatewayConnection{
				Name:     to.StringPtr("test name"),
				Location: to.StringPtr("test location"),
				VirtualNetworkGatewayConnectionPropertiesFormat: &network.VirtualNetworkGatewayConnectionPropertiesFormat{
					ConnectionType: network.Vnet2Vnet,
					VirtualNetworkGateway1: &network.VirtualNetworkGateway{
						ID: to.StringPtr("test gw1 id"),
					},
					VirtualNetworkGateway2: &network.VirtualNetworkGateway{
						ID: to.StringPtr("test gw2 id"),
					},
				},
			},
			true,
		},
		{
			"connection type",
			network.VirtualNetworkGatewayConnection{
				Name: to.StringPtr("test name"),
				VirtualNetworkGatewayConnectionPropertiesFormat: &network.VirtualNetworkGatewayConnectionPropertiesFormat{
					ConnectionType: network.ExpressRoute,
				},
			},
			network.VirtualNetworkGatewayConnection{
				Name:     to.StringPtr("test name"),
				Location: to.StringPtr("test location"),
				VirtualNetworkGatewayConnectionPropertiesFormat: &network.VirtualNetworkGatewayConnectionPropertiesFormat{
					ConnectionType: network.Vnet2Vnet,
					VirtualNetworkGateway1: &network.VirtualNetworkGateway{
						ID: to.StringPtr("test gw1 id"),
					},
					VirtualNetworkGateway2: &network.VirtualNetworkGateway{
						ID: to.StringPtr("test gw2 id"),
					},
				},
			},
			true,
		},
		{
			"vpn gateway 1",
			network.VirtualNetworkGatewayConnection{
				Name: to.StringPtr("test name"),
				VirtualNetworkGatewayConnectionPropertiesFormat: &network.VirtualNetworkGatewayConnectionPropertiesFormat{
					ConnectionType: network.Vnet2Vnet,
					VirtualNetworkGateway1: &network.VirtualNetworkGateway{
						ID: to.StringPtr("wrong gw1 id"),
					},
				},
			},
			network.VirtualNetworkGatewayConnection{
				Name:     to.StringPtr("test name"),
				Location: to.StringPtr("test location"),
				VirtualNetworkGatewayConnectionPropertiesFormat: &network.VirtualNetworkGatewayConnectionPropertiesFormat{
					ConnectionType: network.Vnet2Vnet,
					VirtualNetworkGateway1: &network.VirtualNetworkGateway{
						ID: to.StringPtr("test gw1 id"),
					},
					VirtualNetworkGateway2: &network.VirtualNetworkGateway{
						ID: to.StringPtr("test gw2 id"),
					},
				},
			},
			true,
		},
		{
			"vpn gateway 2",
			network.VirtualNetworkGatewayConnection{
				Name: to.StringPtr("test name"),
				VirtualNetworkGatewayConnectionPropertiesFormat: &network.VirtualNetworkGatewayConnectionPropertiesFormat{
					ConnectionType: network.Vnet2Vnet,
					VirtualNetworkGateway1: &network.VirtualNetworkGateway{
						ID: to.StringPtr("test gw1 id"),
					},
					VirtualNetworkGateway2: &network.VirtualNetworkGateway{
						ID: to.StringPtr("wrong gw2 id"),
					},
				},
			},
			network.VirtualNetworkGatewayConnection{
				Name:     to.StringPtr("test name"),
				Location: to.StringPtr("test location"),
				VirtualNetworkGatewayConnectionPropertiesFormat: &network.VirtualNetworkGatewayConnectionPropertiesFormat{
					ConnectionType: network.Vnet2Vnet,
					VirtualNetworkGateway1: &network.VirtualNetworkGateway{
						ID: to.StringPtr("test gw1 id"),
					},
					VirtualNetworkGateway2: &network.VirtualNetworkGateway{
						ID: to.StringPtr("test gw2 id"),
					},
				},
			},
			true,
		},
		{
			"disconnected",
			network.VirtualNetworkGatewayConnection{
				Name: to.StringPtr("test name"),
				VirtualNetworkGatewayConnectionPropertiesFormat: &network.VirtualNetworkGatewayConnectionPropertiesFormat{
					ConnectionType:   network.Vnet2Vnet,
					ConnectionStatus: network.VirtualNetworkGatewayConnectionStatusNotConnected,
					VirtualNetworkGateway1: &network.VirtualNetworkGateway{
						ID: to.StringPtr("test gw1 id"),
					},
					VirtualNetworkGateway2: &network.VirtualNetworkGateway{
						ID: to.StringPtr("test gw2 id"),
					},
				},
			},
			network.VirtualNetworkGatewayConnection{
				Name:     to.StringPtr("test name"),
				Location: to.StringPtr("test location"),
				VirtualNetworkGatewayConnectionPropertiesFormat: &network.VirtualNetworkGatewayConnectionPropertiesFormat{
					ConnectionType: network.Vnet2Vnet,
					VirtualNetworkGateway1: &network.VirtualNetworkGateway{
						ID: to.StringPtr("test gw1 id"),
					},
					VirtualNetworkGateway2: &network.VirtualNetworkGateway{
						ID: to.StringPtr("test gw2 id"),
					},
				},
			},
			true,
		},
		{
			"connecting",
			network.VirtualNetworkGatewayConnection{
				Name: to.StringPtr("test name"),
				VirtualNetworkGatewayConnectionPropertiesFormat: &network.VirtualNetworkGatewayConnectionPropertiesFormat{
					ConnectionType:   network.Vnet2Vnet,
					ConnectionStatus: network.VirtualNetworkGatewayConnectionStatusConnecting,
					VirtualNetworkGateway1: &network.VirtualNetworkGateway{
						ID: to.StringPtr("test gw1 id"),
					},
					VirtualNetworkGateway2: &network.VirtualNetworkGateway{
						ID: to.StringPtr("test gw2 id"),
					},
				},
			},
			network.VirtualNetworkGatewayConnection{
				Name:     to.StringPtr("test name"),
				Location: to.StringPtr("test location"),
				VirtualNetworkGatewayConnectionPropertiesFormat: &network.VirtualNetworkGatewayConnectionPropertiesFormat{
					ConnectionType: network.Vnet2Vnet,
					VirtualNetworkGateway1: &network.VirtualNetworkGateway{
						ID: to.StringPtr("test gw1 id"),
					},
					VirtualNetworkGateway2: &network.VirtualNetworkGateway{
						ID: to.StringPtr("test gw2 id"),
					},
				},
			},
			false,
		},
		{
			"connection unknown",
			network.VirtualNetworkGatewayConnection{
				Name: to.StringPtr("test name"),
				VirtualNetworkGatewayConnectionPropertiesFormat: &network.VirtualNetworkGatewayConnectionPropertiesFormat{
					ConnectionType:   network.Vnet2Vnet,
					ConnectionStatus: network.VirtualNetworkGatewayConnectionStatusUnknown,
					VirtualNetworkGateway1: &network.VirtualNetworkGateway{
						ID: to.StringPtr("test gw1 id"),
					},
					VirtualNetworkGateway2: &network.VirtualNetworkGateway{
						ID: to.StringPtr("test gw2 id"),
					},
				},
			},
			network.VirtualNetworkGatewayConnection{
				Name:     to.StringPtr("test name"),
				Location: to.StringPtr("test location"),
				VirtualNetworkGatewayConnectionPropertiesFormat: &network.VirtualNetworkGatewayConnectionPropertiesFormat{
					ConnectionType: network.Vnet2Vnet,
					VirtualNetworkGateway1: &network.VirtualNetworkGateway{
						ID: to.StringPtr("test gw1 id"),
					},
					VirtualNetworkGateway2: &network.VirtualNetworkGateway{
						ID: to.StringPtr("test gw2 id"),
					},
				},
			},
			false,
		},
		{
			"connected",
			network.VirtualNetworkGatewayConnection{
				Name: to.StringPtr("test name"),
				VirtualNetworkGatewayConnectionPropertiesFormat: &network.VirtualNetworkGatewayConnectionPropertiesFormat{
					ConnectionType:   network.Vnet2Vnet,
					ConnectionStatus: network.VirtualNetworkGatewayConnectionStatusConnected,
					VirtualNetworkGateway1: &network.VirtualNetworkGateway{
						ID: to.StringPtr("test gw1 id"),
					},
					VirtualNetworkGateway2: &network.VirtualNetworkGateway{
						ID: to.StringPtr("test gw2 id"),
					},
				},
			},
			network.VirtualNetworkGatewayConnection{
				Name:     to.StringPtr("test name"),
				Location: to.StringPtr("test location"),
				VirtualNetworkGatewayConnectionPropertiesFormat: &network.VirtualNetworkGatewayConnectionPropertiesFormat{
					ConnectionType: network.Vnet2Vnet,
					VirtualNetworkGateway1: &network.VirtualNetworkGateway{
						ID: to.StringPtr("test gw1 id"),
					},
					VirtualNetworkGateway2: &network.VirtualNetworkGateway{
						ID: to.StringPtr("test gw2 id"),
					},
				},
			},
			false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			ok := needUpdate(tc.current, tc.desired)
			if ok != tc.expected {
				t.Errorf("expected %t, got %t", tc.expected, ok)
			}
		})
	}
}
