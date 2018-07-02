package network

import (
	"context"
	"fmt"
	"net"
	"testing"

	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/ipam"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/service/controller/setting"
)

func TestComputeSubnets(t *testing.T) {
	testCases := []struct {
		description     string
		network         setting.AzureNetwork
		cidr            string
		expectedSubnets Subnets
		errorMatcher    func(error) bool
	}{
		{
			"ok",
			setting.AzureNetwork{
				MasterSubnetMask: 24,
				VPNSubnetMask:    24,
				WorkerSubnetMask: 24,
			},
			"10.0.0.0/16",
			Subnets{
				Calico: net.IPNet{IP: net.IPv4(10, 0, 128, 0), Mask: net.IPv4Mask(255, 255, 128, 0)},
				Master: net.IPNet{IP: net.IPv4(10, 0, 0, 0), Mask: net.IPv4Mask(255, 255, 255, 0)},
				Parent: net.IPNet{IP: net.IPv4(10, 0, 0, 0), Mask: net.IPv4Mask(255, 255, 0, 0)},
				VPN:    net.IPNet{IP: net.IPv4(10, 0, 2, 0), Mask: net.IPv4Mask(255, 255, 255, 0)},
				Worker: net.IPNet{IP: net.IPv4(10, 0, 1, 0), Mask: net.IPv4Mask(255, 255, 255, 0)},
			},
			nil,
		},
		{
			"cidr too small",
			setting.AzureNetwork{
				MasterSubnetMask: 24,
				VPNSubnetMask:    24,
				WorkerSubnetMask: 24,
			},
			"10.0.0.0/24",
			Subnets{},
			ipam.IsSpaceExhausted,
		},
		{
			"cidr invalid",
			setting.AzureNetwork{
				MasterSubnetMask: 24,
				VPNSubnetMask:    24,
				WorkerSubnetMask: 24,
			},
			"",
			Subnets{},
			func(e error) bool { _, ok := microerror.Cause(e).(*net.ParseError); return ok },
		},
		{
			"subnet mask too big",
			setting.AzureNetwork{
				MasterSubnetMask: 24,
				VPNSubnetMask:    24,
				WorkerSubnetMask: 24,
			},
			"10.17.0.0/16",
			Subnets{},
			ipam.IsSpaceExhausted,
		},
		{
			"subnet mask invalid",
			setting.AzureNetwork{
				MasterSubnetMask: 24,
				VPNSubnetMask:    24,
				WorkerSubnetMask: 0,
			},
			"10.17.0.0/16",
			Subnets{},
			ipam.IsMaskTooBig,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			cr := &providerv1alpha1.AzureConfig{
				Spec: providerv1alpha1.AzureConfigSpec{
					Azure: providerv1alpha1.AzureConfigSpecAzure{
						VirtualNetwork: providerv1alpha1.AzureConfigSpecAzureVirtualNetwork{
							CIDR: tc.cidr,
						},
					},
				},
			}

			ctx := context.TODO()
			subnets, err := ComputeFromCR(ctx, cr, tc.network)

			if tc.errorMatcher != nil {
				if !tc.errorMatcher(err) {
					t.Fatalf("error does not match > %v", err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error > %v", err)
				}

				printSubnets := func(s Subnets) string {
					return fmt.Sprintf("Calico: %s\nMaster: %s\nParent: %s\nVPN: %s\nWorker: %s\n", s.Calico, s.Master, s.Parent, s.VPN, s.Worker)
				}

				printSubnets(*subnets)
				if !subnets.Equal(tc.expectedSubnets) {
					t.Errorf("\ngot\n%s\nexpected\n%s", printSubnets(*subnets), printSubnets(tc.expectedSubnets))
				}
			}
		})
	}
}
