package vpnconnection

import (
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-11-01/network"
	"github.com/Azure/go-autorest/autorest/to"
)

func Test_Resource_VPNGateway_needsUpdate(t *testing.T) {
	testCases := []struct {
		description      string
		current, desired network.VirtualNetworkGatewayConnection
		expected         bool
	}{
		{
			description: "desired state empty",
			current:     network.VirtualNetworkGatewayConnection{},
			desired:     network.VirtualNetworkGatewayConnection{},
			expected:    false,
		},
		{
			description: "name",
			current: network.VirtualNetworkGatewayConnection{
				Name: to.StringPtr("wrong name"),
			},
			desired: network.VirtualNetworkGatewayConnection{
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
			expected: true,
		},
		{
			description: "connection type",
			current: network.VirtualNetworkGatewayConnection{
				Name: to.StringPtr("test name"),
				VirtualNetworkGatewayConnectionPropertiesFormat: &network.VirtualNetworkGatewayConnectionPropertiesFormat{
					ConnectionType: network.ExpressRoute,
				},
			},
			desired: network.VirtualNetworkGatewayConnection{
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
			expected: true,
		},
		{
			description: "vpn gateway 1",
			current: network.VirtualNetworkGatewayConnection{
				Name: to.StringPtr("test name"),
				VirtualNetworkGatewayConnectionPropertiesFormat: &network.VirtualNetworkGatewayConnectionPropertiesFormat{
					ConnectionType: network.Vnet2Vnet,
					VirtualNetworkGateway1: &network.VirtualNetworkGateway{
						ID: to.StringPtr("wrong gw1 id"),
					},
				},
			},
			desired: network.VirtualNetworkGatewayConnection{
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
			expected: true,
		},
		{
			description: "vpn gateway 2",
			current: network.VirtualNetworkGatewayConnection{
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
			desired: network.VirtualNetworkGatewayConnection{
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
			expected: true,
		},
		{
			description: "disconnected",
			current: network.VirtualNetworkGatewayConnection{
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
			desired: network.VirtualNetworkGatewayConnection{
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
			expected: true,
		},
		{
			description: "connecting",
			current: network.VirtualNetworkGatewayConnection{
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
			desired: network.VirtualNetworkGatewayConnection{
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
			expected: false,
		},
		{
			description: "connection unknown",
			current: network.VirtualNetworkGatewayConnection{
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
			desired: network.VirtualNetworkGatewayConnection{
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
			expected: false,
		},
		{
			description: "connected",
			current: network.VirtualNetworkGatewayConnection{
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
			desired: network.VirtualNetworkGatewayConnection{
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
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			ok := needsUpdate(tc.current, tc.desired)
			if ok != tc.expected {
				t.Errorf("expected %t, got %t", tc.expected, ok)
			}
		})
	}
}
