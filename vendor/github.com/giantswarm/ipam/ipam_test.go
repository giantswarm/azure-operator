package ipam

import (
	"bytes"
	"fmt"
	"net"
	"reflect"
	"testing"
)

func ExampleFree() {
	_, network, _ := net.ParseCIDR("10.4.0.0/16")

	firstSubnet, _ := Free(*network, net.CIDRMask(24, 32), []net.IPNet{})
	fmt.Println(firstSubnet.String())

	secondSubnet, _ := Free(*network, net.CIDRMask(28, 32), []net.IPNet{firstSubnet})
	fmt.Println(secondSubnet.String())

	thirdSubnet, _ := Free(*network, net.CIDRMask(24, 32), []net.IPNet{firstSubnet, secondSubnet})
	fmt.Println(thirdSubnet.String())
	// Output:
	// 10.4.0.0/24
	// 10.4.1.0/28
	// 10.4.2.0/24
}

func ExampleHalf() {
	_, network, _ := net.ParseCIDR("10.4.0.0/16")

	firstNetwork, secondNetwork, _ := Half(*network)

	fmt.Println(firstNetwork.String())
	fmt.Println(secondNetwork.String())
	// Output:
	// 10.4.0.0/17
	// 10.4.128.0/17
}

// TestAdd tests the add function.
func TestAdd(t *testing.T) {
	tests := []struct {
		ip         string
		number     int
		expectedIP string
	}{
		{
			ip:         "127.0.0.1",
			number:     0,
			expectedIP: "127.0.0.1",
		},

		{
			ip:         "127.0.0.1",
			number:     1,
			expectedIP: "127.0.0.2",
		},

		{
			ip:         "127.0.0.1",
			number:     2,
			expectedIP: "127.0.0.3",
		},

		{
			ip:         "127.0.0.1",
			number:     -1,
			expectedIP: "127.0.0.0",
		},

		{
			ip:         "0.0.0.0",
			number:     -1,
			expectedIP: "255.255.255.255",
		},

		{
			ip:         "255.255.255.255",
			number:     1,
			expectedIP: "0.0.0.0",
		},
	}

	for index, test := range tests {
		ip := net.ParseIP(test.ip)
		expectedIP := net.ParseIP(test.expectedIP)

		returnedIP := add(ip, test.number)

		if !returnedIP.Equal(expectedIP) {
			t.Fatalf(
				"%v: unexpected ip returned.\nexpected: %v, returned: %v",
				index,
				expectedIP,
				returnedIP,
			)
		}
	}
}

func Test_CalculateSubnetMask(t *testing.T) {
	testCases := []struct {
		name         string
		mask         net.IPMask
		n            uint
		expectedMask net.IPMask
		errorMatcher func(error) bool
	}{
		{
			name:         "case 0: split /24 into one network",
			mask:         net.CIDRMask(24, 32),
			n:            1,
			expectedMask: net.CIDRMask(24, 32),
			errorMatcher: nil,
		},
		{
			name:         "case 1: split /24 into two networks",
			mask:         net.CIDRMask(24, 32),
			n:            2,
			expectedMask: net.CIDRMask(25, 32),
			errorMatcher: nil,
		},
		{
			name:         "case 2: split /24 into three networks",
			mask:         net.CIDRMask(24, 32),
			n:            3,
			expectedMask: net.CIDRMask(26, 32),
			errorMatcher: nil,
		},
		{
			name:         "case 3: split /24 into four networks",
			mask:         net.CIDRMask(24, 32),
			n:            4,
			expectedMask: net.CIDRMask(26, 32),
			errorMatcher: nil,
		},
		{
			name:         "case 4: split /24 into five networks",
			mask:         net.CIDRMask(24, 32),
			n:            5,
			expectedMask: net.CIDRMask(27, 32),
			errorMatcher: nil,
		},
		{
			name:         "case 5: split /22 into 7 networks",
			mask:         net.CIDRMask(22, 32),
			n:            7,
			expectedMask: net.CIDRMask(25, 32),
			errorMatcher: nil,
		},
		{
			name:         "case 6: split /31 into 8 networks (no room)",
			mask:         net.CIDRMask(31, 32),
			n:            7,
			expectedMask: nil,
			errorMatcher: IsInvalidParameter,
		},
		{
			name:         "case 7: IPv6 masks (split /31 for seven networks)",
			mask:         net.CIDRMask(31, 128),
			n:            7,
			expectedMask: net.CIDRMask(34, 128),
			errorMatcher: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mask, err := CalculateSubnetMask(tc.mask, tc.n)

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

			if !reflect.DeepEqual(mask, tc.expectedMask) {
				t.Fatalf("Mask == %q, want %q", mask, tc.expectedMask)
			}
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name             string
		network          string
		subnet           string
		expectedContains bool
	}{
		{
			name:             "case 0: test CIDR contains smaller subnet",
			network:          "10.4.0.0/16",
			subnet:           "10.4.0.0/17",
			expectedContains: true,
		},
		{
			name:             "case 1: test CIDR with IP contains smaller subnet",
			network:          "10.4.0.1/16",
			subnet:           "10.4.0.0/17",
			expectedContains: true,
		},
		{
			name:             "case 2: test CIDR contains itself",
			network:          "10.4.0.0/16",
			subnet:           "10.4.0.0/16",
			expectedContains: true,
		},
		{
			name:             "case 3: test CIDR contains itself but with different representation",
			network:          "10.4.0.0/16",
			subnet:           "10.4.20.0/16",
			expectedContains: true,
		},
		{
			name:             "case 4: test CIDR contains smaller subnet with IP representation",
			network:          "10.4.0.0/16",
			subnet:           "10.4.20.0/17",
			expectedContains: true,
		},
		{
			name:             "case 5: test CIDR doesn't contain subnet that is actually larger than network range",
			network:          "10.4.0.0/16",
			subnet:           "10.4.0.0/15",
			expectedContains: false,
		},
		{
			name:             "case 6: test CIDR doesn't contain different sibling network",
			network:          "10.4.0.0/16",
			subnet:           "10.3.0.0/16",
			expectedContains: false,
		},
		{
			name:             "case 7: test CIDR doesn't contain specific IP from different network (sibling range)",
			network:          "10.4.0.0/16",
			subnet:           "10.3.0.0/32",
			expectedContains: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, network, err := net.ParseCIDR(tc.network)
			if err != nil {
				t.Fatalf("could not parse cidr: %v", tc.network)
			}
			_, subnet, err := net.ParseCIDR(tc.subnet)
			if err != nil {
				t.Fatalf("could not parse cidr: %v", tc.subnet)
			}

			contains := Contains(*network, *subnet)

			if contains != tc.expectedContains {
				t.Errorf("expected contains = %v, got %v", contains, tc.expectedContains)
			}
		})
	}
}

func Test_CanonicalizeSubnets(t *testing.T) {
	testCases := []struct {
		name            string
		network         net.IPNet
		subnets         []net.IPNet
		expectedSubnets []net.IPNet
	}{
		{
			name:            "case 0: deduplicate empty list of subnets",
			network:         mustParseCIDR("192.168.0.0/16"),
			subnets:         []net.IPNet{},
			expectedSubnets: []net.IPNet{},
		},
		{
			name:    "case 1: deduplicate list of subnets with one element",
			network: mustParseCIDR("192.168.0.0/16"),
			subnets: []net.IPNet{
				mustParseCIDR("192.168.2.0/24"),
			},
			expectedSubnets: []net.IPNet{
				mustParseCIDR("192.168.2.0/24"),
			},
		},
		{
			name:    "case 2: deduplicate list of subnets with two non-overlapping elements",
			network: mustParseCIDR("192.168.0.0/16"),
			subnets: []net.IPNet{
				mustParseCIDR("192.168.1.0/24"),
				mustParseCIDR("192.168.2.0/24"),
			},
			expectedSubnets: []net.IPNet{
				mustParseCIDR("192.168.1.0/24"),
				mustParseCIDR("192.168.2.0/24"),
			},
		},
		{
			name:    "case 3: deduplicate list of subnets with two overlapping elements",
			network: mustParseCIDR("192.168.0.0/16"),
			subnets: []net.IPNet{
				mustParseCIDR("192.168.1.0/24"),
				mustParseCIDR("192.168.1.0/24"),
			},
			expectedSubnets: []net.IPNet{
				mustParseCIDR("192.168.1.0/24"),
			},
		},
		{
			name:    "case 4: deduplicate list of subnets with four elements where two overlap",
			network: mustParseCIDR("192.168.0.0/16"),
			subnets: []net.IPNet{
				mustParseCIDR("192.168.1.0/24"),
				mustParseCIDR("192.168.2.0/24"),
				mustParseCIDR("192.168.2.0/24"),
				mustParseCIDR("192.168.3.0/24"),
			},
			expectedSubnets: []net.IPNet{
				mustParseCIDR("192.168.1.0/24"),
				mustParseCIDR("192.168.2.0/24"),
				mustParseCIDR("192.168.3.0/24"),
			},
		},
		{
			name:    "case 5: same as case 4 but with different order",
			network: mustParseCIDR("192.168.0.0/16"),
			subnets: []net.IPNet{
				mustParseCIDR("192.168.1.0/24"),
				mustParseCIDR("192.168.2.0/24"),
				mustParseCIDR("192.168.3.0/24"),
				mustParseCIDR("192.168.3.0/24"),
			},
			expectedSubnets: []net.IPNet{
				mustParseCIDR("192.168.1.0/24"),
				mustParseCIDR("192.168.2.0/24"),
				mustParseCIDR("192.168.3.0/24"),
			},
		},
		{
			name:    "case 6: same as case 4 but with different order",
			network: mustParseCIDR("192.168.0.0/16"),
			subnets: []net.IPNet{
				mustParseCIDR("192.168.1.0/24"),
				mustParseCIDR("192.168.1.0/24"),
				mustParseCIDR("192.168.2.0/24"),
				mustParseCIDR("192.168.3.0/24"),
			},
			expectedSubnets: []net.IPNet{
				mustParseCIDR("192.168.1.0/24"),
				mustParseCIDR("192.168.2.0/24"),
				mustParseCIDR("192.168.3.0/24"),
			},
		},
		{
			name:    "case 7: same as case 4 but with different order",
			network: mustParseCIDR("192.168.0.0/16"),
			subnets: []net.IPNet{
				mustParseCIDR("192.168.1.0/24"),
				mustParseCIDR("192.168.2.0/24"),
				mustParseCIDR("192.168.3.0/24"),
				mustParseCIDR("192.168.1.0/24"),
			},
			expectedSubnets: []net.IPNet{
				mustParseCIDR("192.168.1.0/24"),
				mustParseCIDR("192.168.2.0/24"),
				mustParseCIDR("192.168.3.0/24"),
			},
		},
		{
			name:    "case 7: deduplicate list of subnets with fiveelements where two overlap",
			network: mustParseCIDR("192.168.0.0/16"),
			subnets: []net.IPNet{
				mustParseCIDR("192.168.1.0/24"),
				mustParseCIDR("192.168.2.0/24"),
				mustParseCIDR("192.168.3.0/24"),
				mustParseCIDR("192.168.1.0/24"),
				mustParseCIDR("192.168.4.0/24"),
			},
			expectedSubnets: []net.IPNet{
				mustParseCIDR("192.168.1.0/24"),
				mustParseCIDR("192.168.2.0/24"),
				mustParseCIDR("192.168.3.0/24"),
				mustParseCIDR("192.168.4.0/24"),
			},
		},
		{
			name:    "case 8: deduplicate list of subnets with duplicates and IPs from different segments",
			network: mustParseCIDR("192.168.0.0/16"),
			subnets: []net.IPNet{
				mustParseCIDR("192.168.1.0/24"),
				mustParseCIDR("192.168.2.0/24"),
				mustParseCIDR("192.168.3.0/24"),
				mustParseCIDR("192.168.1.0/24"),
				mustParseCIDR("192.168.4.0/24"),
				mustParseCIDR("172.31.0.1/16"),
				mustParseCIDR("10.2.0.4/24"),
			},
			expectedSubnets: []net.IPNet{
				mustParseCIDR("192.168.1.0/24"),
				mustParseCIDR("192.168.2.0/24"),
				mustParseCIDR("192.168.3.0/24"),
				mustParseCIDR("192.168.4.0/24"),
			},
		},
		{
			name:    "case 9: ensure that smaller overlapping subnets are accepted",
			network: mustParseCIDR("10.0.0.0/8"),
			subnets: []net.IPNet{
				mustParseCIDR("10.1.0.0/16"),
				mustParseCIDR("10.1.0.0/24"),
				mustParseCIDR("10.1.1.0/24"),
			},
			expectedSubnets: []net.IPNet{
				mustParseCIDR("10.1.0.0/16"),
				mustParseCIDR("10.1.0.0/24"),
				mustParseCIDR("10.1.1.0/24"),
			},
		},
		{
			name:    "case 10: same as case 9 but with different input ordering",
			network: mustParseCIDR("10.0.0.0/8"),
			subnets: []net.IPNet{
				mustParseCIDR("10.1.0.0/24"),
				mustParseCIDR("10.1.1.0/24"),
				mustParseCIDR("10.1.0.0/16"),
			},
			expectedSubnets: []net.IPNet{
				mustParseCIDR("10.1.0.0/24"),
				mustParseCIDR("10.1.1.0/24"),
				mustParseCIDR("10.1.0.0/16"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			subnets := CanonicalizeSubnets(tc.network, tc.subnets)

			if !reflect.DeepEqual(subnets, tc.expectedSubnets) {
				msg := "expected: {\n"
				for _, n := range tc.expectedSubnets {
					msg += fmt.Sprintf("\t%s,\n", n.String())
				}
				msg += "}\n\ngot: {\n"
				for _, n := range subnets {
					msg += fmt.Sprintf("\t%s,\n", n.String())
				}
				msg += "}"
				t.Fatal(msg)
			}
		})
	}
}

// TestDecimalToIP tests the decimalToIP function.
func TestDecimalToIP(t *testing.T) {
	tests := []struct {
		decimal    int
		expectedIP string
	}{
		{
			decimal:    0,
			expectedIP: "0.0.0.0",
		},
		{
			decimal:    1283,
			expectedIP: "0.0.5.3",
		},
		{
			decimal:    167772160,
			expectedIP: "10.0.0.0",
		},
		{
			decimal:    168034304,
			expectedIP: "10.4.0.0",
		},
		{
			decimal:    4294967295,
			expectedIP: "255.255.255.255",
		},
	}

	for index, test := range tests {
		returnedIP := decimalToIP(test.decimal)
		expectedIP := net.ParseIP(test.expectedIP)

		if !returnedIP.Equal(expectedIP) {
			t.Fatalf(
				"%v: unexpected decimal returned.\nexpected: %v, returned: %v",
				index,
				expectedIP,
				returnedIP,
			)
		}
	}
}

// TestFree tests the Free function.
func TestFree(t *testing.T) {
	tests := []struct {
		network              string
		mask                 int
		subnets              []string
		expectedNetwork      string
		expectedErrorHandler func(error) bool
	}{
		// Test that a network with no existing subnets returns the correct subnet.
		{
			network:         "10.4.0.0/16",
			mask:            24,
			subnets:         []string{},
			expectedNetwork: "10.4.0.0/24",
		},

		// Test that a network with one existing subnet returns the correct subnet.
		{
			network:         "10.4.0.0/16",
			mask:            24,
			subnets:         []string{"10.4.0.0/24"},
			expectedNetwork: "10.4.1.0/24",
		},

		// Test that a network with two existing (non-fragmented) subnets returns the correct subnet.
		{
			network:         "10.4.0.0/16",
			mask:            24,
			subnets:         []string{"10.4.0.0/24", "10.4.1.0/24"},
			expectedNetwork: "10.4.2.0/24",
		},

		// Test that a network with an existing subnet, that is fragmented,
		// and can fit one network before, returns the correct subnet.
		{
			network:         "10.4.0.0/16",
			mask:            24,
			subnets:         []string{"10.4.1.0/24"},
			expectedNetwork: "10.4.0.0/24",
		},

		// Test that a network with an existing subnet, that is fragmented,
		// and can fit one network before, returns the correct subnet,
		// given a smaller mask.
		{
			network:         "10.4.0.0/16",
			mask:            25,
			subnets:         []string{"10.4.1.0/24"},
			expectedNetwork: "10.4.0.0/25",
		},

		// Test that a network with an existing subnet, that is fragmented,
		// but can't fit the requested network size before, returns the correct subnet.
		{
			network:         "10.4.0.0/16",
			mask:            23,
			subnets:         []string{"10.4.1.0/24"}, // 10.4.1.0 - 10.4.1.255
			expectedNetwork: "10.4.2.0/23",           // 10.4.2.0 - 10.4.3.255
		},

		// Test that a network with no existing subnets returns the correct subnet,
		// for a mask that does not fall on an octet boundary.
		{
			network:         "10.4.0.0/24",
			mask:            26,
			subnets:         []string{},
			expectedNetwork: "10.4.0.0/26",
		},

		// Test that a network with one existing subnet returns the correct subnet,
		// for a mask that does not fall on an octet boundary.
		{
			network:         "10.4.0.0/24",
			mask:            26,
			subnets:         []string{"10.4.0.0/26"},
			expectedNetwork: "10.4.0.64/26",
		},

		// Test that a network with two existing fragmented subnets,
		// with a mask that does not fall on an octet boundary, returns the correct subnet.
		{
			network:         "10.4.0.0/24",
			mask:            26,
			subnets:         []string{"10.4.0.0/26", "10.4.0.128/26"},
			expectedNetwork: "10.4.0.64/26",
		},

		// Test a setup with multiple, fragmented networks, of different sizes.
		{
			network: "10.4.0.0/24",
			mask:    29,
			subnets: []string{
				"10.4.0.0/26",
				"10.4.0.64/28",
				"10.4.0.80/28",
				"10.4.0.112/28",
				"10.4.0.128/26",
			},
			expectedNetwork: "10.4.0.96/29",
		},

		// Test where a network the same size as the main network is requested.
		{
			network:         "10.4.0.0/16",
			mask:            16,
			subnets:         []string{},
			expectedNetwork: "10.4.0.0/16",
		},

		// Test a setup where a network larger than the main network is requested.
		{
			network:              "10.4.0.0/16",
			mask:                 15,
			subnets:              []string{},
			expectedErrorHandler: IsMaskTooBig,
		},

		// Test where the existing networks are not ordered.
		{
			network:         "10.4.0.0/16",
			mask:            24,
			subnets:         []string{"10.4.1.0/24", "10.4.0.0/24"},
			expectedNetwork: "10.4.2.0/24",
		},

		// Test where the existing networks are fragmented, and not ordered.
		{
			network:         "10.4.0.0/16",
			mask:            24,
			subnets:         []string{"10.4.2.0/24", "10.4.0.0/24"},
			expectedNetwork: "10.4.1.0/24",
		},

		// Test where the range is full.
		{
			network:              "10.4.0.0/16",
			mask:                 17,
			subnets:              []string{"10.4.0.0/17", "10.4.128.0/17"},
			expectedErrorHandler: IsSpaceExhausted,
		},

		// Test where the subnet is not within the network.
		{
			network:              "10.4.0.0/16",
			mask:                 24,
			subnets:              []string{"10.5.0.0/24"},
			expectedErrorHandler: IsIPNotContained,
		},

		{
			network: "10.1.0.0/16",
			mask:    24,
			subnets: []string{
				"10.1.1.0/26",
				"10.1.1.64/26",
				"10.1.1.128/26",
				"10.1.1.192/26",
				"10.1.0.0/25",
				"10.1.0.128/25",
				"10.1.0.0/24",
				"10.1.2.0/27",
				"10.1.2.32/27",
				"10.1.2.64/27",
				"10.1.2.96/27",
				"10.1.2.128/27",
				"10.1.2.160/27",
				"10.1.2.0/24",
			},
			expectedErrorHandler: nil,
			expectedNetwork:      "10.1.3.0/24",
		},
	}

	for index, test := range tests {
		_, network, err := net.ParseCIDR(test.network)
		if err != nil {
			t.Fatalf("%v: could not parse cidr: %v", index, test.network)
		}

		mask := net.CIDRMask(test.mask, 32)

		subnets := []net.IPNet{}
		for _, e := range test.subnets {
			_, n, err := net.ParseCIDR(e)
			if err != nil {
				t.Fatalf("%v: could not parse cidr: %v", index, test.network)
			}
			subnets = append(subnets, *n)
		}

		_, expectedNetwork, _ := net.ParseCIDR(test.expectedNetwork)

		returnedNetwork, err := Free(*network, mask, subnets)

		if err != nil {
			if test.expectedErrorHandler == nil {
				t.Fatalf("%v: unexpected error returned.\nreturned: %v", index, err)
			}
			if !test.expectedErrorHandler(err) {
				t.Fatalf("%v: incorrect error returned.\nreturned: %v", index, err)
			}
		} else {
			if test.expectedErrorHandler != nil {
				t.Fatalf("%v: expected error not returned.", index)
			}

			if !ipNetEqual(returnedNetwork, *expectedNetwork) {
				t.Fatalf(
					"%v: unexpected network returned. \nexpected: %s (%#v, %#v) \nreturned: %s (%#v, %#v)",
					index,

					expectedNetwork.String(),
					expectedNetwork.IP,
					expectedNetwork.Mask,

					returnedNetwork.String(),
					returnedNetwork.IP,
					returnedNetwork.Mask,
				)
			}
		}
	}
}

// TestFreeIPRanges tests the freeIPRanges function.
func TestFreeIPRanges(t *testing.T) {
	tests := []struct {
		network              string
		subnets              []string
		expectedFreeIPRanges []ipRange
		expectedErrorHandler func(error) bool
	}{
		// Test that given a network with no subnets,
		// the entire network is returned as a free range.
		{
			network: "10.4.0.0/16",
			subnets: []string{},
			expectedFreeIPRanges: []ipRange{
				ipRange{
					start: net.ParseIP("10.4.0.0"),
					end:   net.ParseIP("10.4.255.255"),
				},
			},
		},

		// Test that given a network, and one subnet at the start of the network,
		// the entire remaining network - that is, the network minus the subnet,
		// is returned as a free range.
		{
			network: "10.4.0.0/16",
			subnets: []string{"10.4.0.0/24"},
			expectedFreeIPRanges: []ipRange{
				ipRange{
					start: net.ParseIP("10.4.1.0"),
					end:   net.ParseIP("10.4.255.255"),
				},
			},
		},

		// Test that given a network, and two contiguous subnets,
		// the entire remaining network (afterwards) is returned as a free range.
		{
			network: "10.4.0.0/16",
			subnets: []string{"10.4.0.0/24", "10.4.1.0/24"},
			expectedFreeIPRanges: []ipRange{
				ipRange{
					start: net.ParseIP("10.4.2.0"),
					end:   net.ParseIP("10.4.255.255"),
				},
			},
		},

		// Test that given a network, and one fragmented subnet,
		// the two remaining free ranges are returned as free.
		{
			network: "10.4.0.0/16",
			subnets: []string{"10.4.1.0/24"},
			expectedFreeIPRanges: []ipRange{
				ipRange{
					start: net.ParseIP("10.4.0.0"),
					end:   net.ParseIP("10.4.0.255"),
				},
				ipRange{
					start: net.ParseIP("10.4.2.0"),
					end:   net.ParseIP("10.4.255.255"),
				},
			},
		},

		// Test that given a network, and three fragmented subnets,
		// the 4 remaining free ranges are returned as free.
		{
			network: "10.4.0.0/16",
			subnets: []string{
				"10.4.10.0/24",
				"10.4.12.0/24",
				"10.4.14.0/24",
			},
			expectedFreeIPRanges: []ipRange{
				ipRange{
					start: net.ParseIP("10.4.0.0"),
					end:   net.ParseIP("10.4.9.255"),
				},
				ipRange{
					start: net.ParseIP("10.4.11.0"),
					end:   net.ParseIP("10.4.11.255"),
				},
				ipRange{
					start: net.ParseIP("10.4.13.0"),
					end:   net.ParseIP("10.4.13.255"),
				},
				ipRange{
					start: net.ParseIP("10.4.15.0"),
					end:   net.ParseIP("10.4.255.255"),
				},
			},
		},
	}

	for index, test := range tests {
		_, network, err := net.ParseCIDR(test.network)
		if err != nil {
			t.Fatalf("%v: could not parse cidr: %v", index, test.network)
		}

		subnets := []net.IPNet{}
		for _, subnetString := range test.subnets {
			_, subnet, err := net.ParseCIDR(subnetString)
			if err != nil {
				t.Fatalf("%v: could not parse cidr: %v", index, subnetString)
			}

			subnets = append(subnets, *subnet)
		}

		freeSubnets, err := freeIPRanges(*network, subnets)

		if err != nil {
			if test.expectedErrorHandler == nil {
				t.Fatalf("%v: unexpected error returned.\nreturned: %v", index, err)
			}
			if !test.expectedErrorHandler(err) {
				t.Fatalf("%v: incorrect error returned.\nreturned: %v", index, err)
			}
		} else {
			if test.expectedErrorHandler != nil {
				t.Fatalf("%v: expected error not returned.", index)
			}

			if !ipRangesEqual(freeSubnets, test.expectedFreeIPRanges) {
				t.Fatalf(
					"%v: unexpected free subnets returned.\nexpected: %v\nreturned: %v",
					index,
					test.expectedFreeIPRanges,
					freeSubnets,
				)
			}
		}
	}
}

func TestHalf(t *testing.T) {
	tests := []struct {
		Network              string
		ExpectedFirst        string
		ExpectedSecond       string
		ExpectedErrorMatcher func(error) bool
	}{
		// Test 0.
		{
			Network:              "10.4.0.0/16",
			ExpectedFirst:        "10.4.0.0/17",
			ExpectedSecond:       "10.4.128.0/17",
			ExpectedErrorMatcher: nil,
		},
		// Test 1.
		{
			Network:              "10.4.0.0/32",
			ExpectedFirst:        "<nil>",
			ExpectedSecond:       "<nil>",
			ExpectedErrorMatcher: IsMaskTooBig,
		},
		// Test 2.
		{
			Network:              "10.4.0.0/31",
			ExpectedFirst:        "10.4.0.0/32",
			ExpectedSecond:       "10.4.0.1/32",
			ExpectedErrorMatcher: nil,
		},
	}

	for i, tc := range tests {
		_, network, err := net.ParseCIDR(tc.Network)
		if err != nil {
			t.Fatalf("test %d: could not parse cidr: %v", i, tc.Network)
		}

		first, second, err := Half(*network)

		if tc.ExpectedErrorMatcher == nil && err != nil {
			t.Errorf("test %d: unexpected err = %#v", i, err)
		}
		if tc.ExpectedErrorMatcher != nil && !tc.ExpectedErrorMatcher(err) {
			t.Errorf("test %d: unexpected err = %#v", i, err)
		}

		if first.String() != tc.ExpectedFirst {
			t.Errorf("test %d: expected first = %q, got %q", i, first.String(), tc.ExpectedFirst)
		}
		if second.String() != tc.ExpectedSecond {
			t.Errorf("test %d: expected second = %q, got %q", i, second.String(), tc.ExpectedSecond)
		}
	}
}

// TestIPToDecimal tests the ipToDecimal function.
func TestIPToDecimal(t *testing.T) {
	tests := []struct {
		ip              string
		expectedDecimal int
	}{
		{
			ip:              "0.0.0.0",
			expectedDecimal: 0,
		},
		{
			ip:              "0.0.05.3",
			expectedDecimal: 1283,
		},
		{
			ip:              "0.0.5.3",
			expectedDecimal: 1283,
		},
		{
			ip:              "10.0.0.0",
			expectedDecimal: 167772160,
		},
		{
			ip:              "10.4.0.0",
			expectedDecimal: 168034304,
		},
		{
			ip:              "255.255.255.255",
			expectedDecimal: 4294967295,
		},
	}

	for index, test := range tests {
		returnedDecimal := ipToDecimal(net.ParseIP(test.ip))

		if returnedDecimal != test.expectedDecimal {
			t.Fatalf(
				"%v: unexpected decimal returned.\nexpected: %v, returned: %v",
				index,
				test.expectedDecimal,
				returnedDecimal,
			)
		}
	}
}

// TestNewIPRange tests the newIPRange function.
func TestNewIPRange(t *testing.T) {
	tests := []struct {
		network         string
		expectedIPRange ipRange
	}{
		{
			network: "0.0.0.0/0",
			expectedIPRange: ipRange{
				start: net.ParseIP("0.0.0.0").To4(),
				end:   net.ParseIP("255.255.255.255").To4(),
			},
		},

		{
			network: "10.4.0.0/8",
			expectedIPRange: ipRange{
				start: net.ParseIP("10.0.0.0").To4(),
				end:   net.ParseIP("10.255.255.255").To4(),
			},
		},

		{
			network: "10.4.0.0/16",
			expectedIPRange: ipRange{
				start: net.ParseIP("10.4.0.0").To4(),
				end:   net.ParseIP("10.4.255.255").To4(),
			},
		},

		{
			network: "10.4.0.0/24",
			expectedIPRange: ipRange{
				start: net.ParseIP("10.4.0.0").To4(),
				end:   net.ParseIP("10.4.0.255").To4(),
			},
		},

		{
			network: "172.168.0.0/25",
			expectedIPRange: ipRange{
				start: net.ParseIP("172.168.0.0").To4(),
				end:   net.ParseIP("172.168.0.127").To4(),
			},
		},
	}

	for index, test := range tests {
		_, network, _ := net.ParseCIDR(test.network)
		ipRange := newIPRange(*network)

		if !reflect.DeepEqual(ipRange, test.expectedIPRange) {
			t.Fatalf(
				"%v: unexpected ipRange returned.\nexpected: %#v\nreturned: %#v\n",
				index,
				test.expectedIPRange,
				ipRange,
			)
		}
	}
}

// TestSize tests the Size function.
func TestSize(t *testing.T) {
	tests := []struct {
		mask         int
		expectedSize int
	}{
		{
			mask:         0,
			expectedSize: 4294967296,
		},
		{
			mask:         1,
			expectedSize: 2147483648,
		},
		{
			mask:         23,
			expectedSize: 512,
		},
		{
			mask:         24,
			expectedSize: 256,
		},
		{
			mask:         25,
			expectedSize: 128,
		},
		{
			mask:         31,
			expectedSize: 2,
		},
		{
			mask:         32,
			expectedSize: 1,
		},
	}

	for index, test := range tests {
		returnedSize := size(net.CIDRMask(test.mask, 32))

		if returnedSize != test.expectedSize {
			t.Fatalf(
				"%v: unexpected size returned.\nexpected: %v, returned: %v",
				index,
				test.expectedSize,
				returnedSize,
			)
		}
	}
}

// TestSpace tests the space function.
func TestSpace(t *testing.T) {
	tests := []struct {
		freeIPRanges         []ipRange
		mask                 int
		expectedIP           net.IP
		expectedErrorHandler func(error) bool
	}{
		// Test a case of fitting a network into an unused network.
		{
			freeIPRanges: []ipRange{
				{
					start: net.ParseIP("10.4.0.0"),
					end:   net.ParseIP("10.4.255.255"),
				},
			},
			mask:       24,
			expectedIP: net.ParseIP("10.4.0.0"),
		},

		// Test fitting a network into a network, with one subnet,
		// at the start of the range.
		{
			freeIPRanges: []ipRange{
				{
					start: net.ParseIP("10.4.1.0"),
					end:   net.ParseIP("10.4.255.255"),
				},
			},
			mask:       24,
			expectedIP: net.ParseIP("10.4.1.0"),
		},

		// Test adding a network that fills the range
		{
			freeIPRanges: []ipRange{
				{
					start: net.ParseIP("10.4.0.0"),
					end:   net.ParseIP("10.4.255.255"),
				},
			},
			mask:       16,
			expectedIP: net.ParseIP("10.4.0.0"),
		},

		// Test adding a network that is too large.
		{
			freeIPRanges: []ipRange{
				{
					start: net.ParseIP("10.4.0.0"),
					end:   net.ParseIP("10.4.255.255"),
				},
			},
			mask:                 15,
			expectedErrorHandler: IsSpaceExhausted,
		},

		// Test adding a slightly larger network,
		// given a smaller, non-contiguous subnet.
		{
			freeIPRanges: []ipRange{
				{
					start: net.ParseIP("10.4.0.0"),
					end:   net.ParseIP("10.4.0.255"),
				},
				{
					start: net.ParseIP("10.4.2.0"),
					end:   net.ParseIP("10.4.255.255"),
				},
			},
			mask:       23,
			expectedIP: net.ParseIP("10.4.2.0"),
		},

		// Test allocating /24 network when the first free range starts from /26.
		{
			freeIPRanges: []ipRange{
				{
					start: net.ParseIP("10.1.2.192"),
					end:   net.ParseIP("10.1.255.255"),
				},
			},
			mask:       24,
			expectedIP: net.ParseIP("10.1.3.0"),
		},
	}

	for index, test := range tests {
		mask := net.CIDRMask(test.mask, 32)

		ip, err := space(test.freeIPRanges, mask)

		if err != nil {
			if test.expectedErrorHandler == nil {
				t.Fatalf("%v: unexpected error returned.\nreturned: %v", index, err)
			}
			if !test.expectedErrorHandler(err) {
				t.Fatalf("%v: incorrect error returned.\nreturned: %v", index, err)
			}
		} else {
			if test.expectedErrorHandler != nil {
				t.Fatalf("%v: expected error not returned.", index)
			}

			if !ip.Equal(test.expectedIP) {
				t.Fatalf(
					"%v: unexpected ip returned. \nexpected: %v\nreturned: %v",
					index,
					test.expectedIP,
					ip,
				)
			}
		}
	}
}

func Test_Split(t *testing.T) {
	testCases := []struct {
		name            string
		network         net.IPNet
		n               uint
		expectedSubnets []net.IPNet
		errorMatcher    func(error) bool
	}{
		{
			name:    "case 0: split /24 into four networks",
			network: mustParseCIDR("192.168.8.0/24"),
			n:       4,
			expectedSubnets: []net.IPNet{
				mustParseCIDR("192.168.8.0/26"),
				mustParseCIDR("192.168.8.64/26"),
				mustParseCIDR("192.168.8.128/26"),
				mustParseCIDR("192.168.8.192/26"),
			},
			errorMatcher: nil,
		},
		{
			name:    "case 1: split /22 into 7 networks",
			network: mustParseCIDR("10.100.0.0/22"),
			n:       7,
			expectedSubnets: []net.IPNet{
				mustParseCIDR("10.100.0.0/25"),
				mustParseCIDR("10.100.0.128/25"),
				mustParseCIDR("10.100.1.0/25"),
				mustParseCIDR("10.100.1.128/25"),
				mustParseCIDR("10.100.2.0/25"),
				mustParseCIDR("10.100.2.128/25"),
				mustParseCIDR("10.100.3.0/25"),
			},
			errorMatcher: nil,
		},
		{
			name:            "case 2: split /31 into 8 networks (no room)",
			network:         mustParseCIDR("192.168.8.128/31"),
			n:               7,
			expectedSubnets: nil,
			errorMatcher:    IsInvalidParameter,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			subnets, err := Split(tc.network, tc.n)

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

			if !reflect.DeepEqual(subnets, tc.expectedSubnets) {
				msg := "expected: {\n"
				for _, n := range tc.expectedSubnets {
					msg += fmt.Sprintf("\t%s,\n", n.String())
				}
				msg += "}\n\ngot: {\n"
				for _, n := range subnets {
					msg += fmt.Sprintf("\t%s,\n", n.String())
				}
				msg += "}"
				t.Fatal(msg)

			}
		})
	}
}

func mustParseCIDR(val string) net.IPNet {
	_, n, err := net.ParseCIDR(val)
	if err != nil {
		panic(err)
	}

	return *n
}

// ipNetEqual returns true if the given IPNets refer to the same network.
func ipNetEqual(a, b net.IPNet) bool {
	return a.IP.Equal(b.IP) && bytes.Equal(a.Mask, b.Mask)
}

// ipRangesEqual returns true if both given ipRanges are equal, false otherwise.
func ipRangesEqual(a, b []ipRange) bool {
	if len(a) != len(b) {
		return false
	}

	for i := 0; i < len(a); i++ {
		if !a[i].start.Equal(b[i].start) {
			return false
		}

		if !a[i].end.Equal(b[i].end) {
			return false
		}
	}

	return true
}
