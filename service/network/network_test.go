package network

import (
	"fmt"
	"net"
	"testing"

	"github.com/giantswarm/ipam"
)

func TestComputeSubnets(t *testing.T) {
	testCases := []struct {
		description     string
		cidr            string
		expectedSubnets Subnets
		errorMatcher    func(error) bool
	}{
		{
			"ok",
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
			"10.0.0.0/24",
			Subnets{},
			ipam.IsSpaceExhausted,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			_, vnet, err := net.ParseCIDR(tc.cidr)
			if err != nil {
				t.Fatalf("expected %#v got %#v", nil, err)
			}
			subnets, err := Compute(*vnet)
			if tc.errorMatcher != nil {
				if !tc.errorMatcher(err) {
					t.Fatalf("expected %#v got %#v", true, false)
				}
			} else {
				if err != nil {
					t.Fatalf("expected %#v got %#v", nil, err)
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
