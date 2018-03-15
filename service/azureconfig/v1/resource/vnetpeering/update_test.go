package vnetpeering

import (
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2017-09-01/network"
	"github.com/Azure/go-autorest/autorest/to"
)

func TestNeedUpdate(t *testing.T) {
	cases := []struct {
		current  network.VirtualNetworkPeering
		desired  network.VirtualNetworkPeering
		expected bool
		isError  bool
	}{
		{
			// Empty desired produce an error
			network.VirtualNetworkPeering{},
			network.VirtualNetworkPeering{},
			false,
			true,
		},
		{
			network.VirtualNetworkPeering{
				// current with additional values does not need update
				Name: to.StringPtr("some Name"),
				VirtualNetworkPeeringPropertiesFormat: &network.VirtualNetworkPeeringPropertiesFormat{
					AllowVirtualNetworkAccess: to.BoolPtr(true),
					AllowForwardedTraffic:     to.BoolPtr(false),
					AllowGatewayTransit:       to.BoolPtr(false),
					UseRemoteGateways:         to.BoolPtr(false),
					RemoteVirtualNetwork: &network.SubResource{
						ID: to.StringPtr("some ID"),
					},
					RemoteAddressSpace: &network.AddressSpace{
						AddressPrefixes: &[]string{
							"10.0.0.0/16",
						},
					},
					PeeringState:      "some peering state",
					ProvisioningState: to.StringPtr("some provisioning state"),
				},
			},
			network.VirtualNetworkPeering{
				Name: to.StringPtr("some Name"),
				VirtualNetworkPeeringPropertiesFormat: &network.VirtualNetworkPeeringPropertiesFormat{
					AllowVirtualNetworkAccess: to.BoolPtr(true),
					RemoteVirtualNetwork: &network.SubResource{
						ID: to.StringPtr("some ID"),
					},
				},
			},
			false,
			false,
		},
		{
			network.VirtualNetworkPeering{
				// need an update when RemoteVirtualNetwork.ID property change
				Name: to.StringPtr("some Name"),
				VirtualNetworkPeeringPropertiesFormat: &network.VirtualNetworkPeeringPropertiesFormat{
					AllowVirtualNetworkAccess: to.BoolPtr(true),
					RemoteVirtualNetwork: &network.SubResource{
						ID: to.StringPtr("some other ID"),
					},
				},
			},
			network.VirtualNetworkPeering{
				Name: to.StringPtr("some Name"),
				VirtualNetworkPeeringPropertiesFormat: &network.VirtualNetworkPeeringPropertiesFormat{
					AllowVirtualNetworkAccess: to.BoolPtr(true),
					RemoteVirtualNetwork: &network.SubResource{
						ID: to.StringPtr("some ID"),
					},
				},
			},
			true,
			false,
		},
		{
			network.VirtualNetworkPeering{
				// need an update when AllowVirtualNetworkAccess property change
				Name: to.StringPtr("some Name"),
				VirtualNetworkPeeringPropertiesFormat: &network.VirtualNetworkPeeringPropertiesFormat{
					AllowVirtualNetworkAccess: to.BoolPtr(false),
					RemoteVirtualNetwork: &network.SubResource{
						ID: to.StringPtr("some ID"),
					},
				},
			},
			network.VirtualNetworkPeering{
				Name: to.StringPtr("some Name"),
				VirtualNetworkPeeringPropertiesFormat: &network.VirtualNetworkPeeringPropertiesFormat{
					AllowVirtualNetworkAccess: to.BoolPtr(true),
					RemoteVirtualNetwork: &network.SubResource{
						ID: to.StringPtr("some ID"),
					},
				},
			},
			true,
			false,
		},
		{
			network.VirtualNetworkPeering{
				// need an update when Name property change
				Name: to.StringPtr("some Name"),
				VirtualNetworkPeeringPropertiesFormat: &network.VirtualNetworkPeeringPropertiesFormat{
					AllowVirtualNetworkAccess: to.BoolPtr(true),
					RemoteVirtualNetwork: &network.SubResource{
						ID: to.StringPtr("some ID"),
					},
				},
			},
			network.VirtualNetworkPeering{
				Name: to.StringPtr("some other Name"),
				VirtualNetworkPeeringPropertiesFormat: &network.VirtualNetworkPeeringPropertiesFormat{
					AllowVirtualNetworkAccess: to.BoolPtr(true),
					RemoteVirtualNetwork: &network.SubResource{
						ID: to.StringPtr("some ID"),
					},
				},
			},
			true,
			false,
		},
	}

	for _, c := range cases {
		result, err := needUpdate(c.current, c.desired)
		if c.isError {
			if err == nil {
				t.Errorf("expected error got '%t'", err)
			}
		} else {
			if result != c.expected {
				t.Errorf("expected '%t' got '%t'", c.expected, result)
			}
		}
	}
}
