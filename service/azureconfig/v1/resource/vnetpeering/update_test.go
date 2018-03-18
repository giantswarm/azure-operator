package vnetpeering

import (
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2017-09-01/network"
	"github.com/Azure/go-autorest/autorest/to"
)

func TestNeedUpdate(t *testing.T) {
	testCases := []struct {
		name         string
		current      network.VirtualNetworkPeering
		desired      network.VirtualNetworkPeering
		expected     bool
		errorMatcher func(err error) bool
	}{
		{
			"case 0: empty desired state returns invalidDesired error",
			network.VirtualNetworkPeering{},
			network.VirtualNetworkPeering{},
			false,
			IsInvalidDesiredState,
		},
		{
			"case 1: current state with additional values does not need update",
			network.VirtualNetworkPeering{
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
					PeeringState:      network.Connected,
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
			nil,
		},
		{
			"case 2: need an update when RemoteVirtualNetwork.ID property change",
			network.VirtualNetworkPeering{
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
			nil,
		},
		{
			"case 3: need an update when AllowVirtualNetworkAccess property change",
			network.VirtualNetworkPeering{
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
			nil,
		},
		{
			"case 4: need an update when Name property change",
			network.VirtualNetworkPeering{
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
			nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ok, err := needUpdate(tc.current, tc.desired)

			switch {
			case err == nil && tc.errorMatcher == nil:
				// correct; carry on
			case err != nil && tc.errorMatcher == nil:
				t.Fatalf("error == %#v, want nil", err)
			case err == nil && tc.errorMatcher != nil:
				t.Fatalf("error == nil, want non-nil")
			case !tc.errorMatcher(err):
				t.Fatalf("error == %#v, want matching", err)
			}

			if ok != tc.expected {
				t.Fatalf("ok == %v, want %v", ok, tc.expected)
			}
		})
	}
}
