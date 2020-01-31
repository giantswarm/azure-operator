package ipam

import (
	"context"
	"fmt"
	"net"

	"github.com/giantswarm/micrologger/microloggertest"
	"github.com/giantswarm/microstorage/memory"
)

// This example demonstrates how the IPAM service can be used to dynamically
// manage subnets within a larger network.
func Example() {
	ctx := context.Background()

	// Construct a new IPAM service.
	logger := microloggertest.New()
	storage, _ := memory.New(memory.Config{})
	_, network, _ := net.ParseCIDR("10.4.0.0/16")

	config := Config{
		Logger:  logger,
		Storage: storage,
		Network: network,
	}

	service, _ := New(config)

	// Request a subnet. There are no subnets in the network currently,
	// so this subnet will be at the start of the IP range.
	firstNetwork, _ := service.CreateSubnet(ctx, net.CIDRMask(24, 32), "arbitrary value", nil)
	fmt.Println(firstNetwork.String())

	// Request a second, smaller subnet.
	// There is one subnet currently, so this subnet will begin after
	// the previous subnet.
	secondNetwork, _ := service.CreateSubnet(ctx, net.CIDRMask(32, 32), "", nil)
	fmt.Println(secondNetwork.String())

	// Release the first subnet from the service.
	// This makes the IP range available for future operations.
	service.DeleteSubnet(ctx, firstNetwork)

	// Request a third subnet.
	// As the range at the start of the network is free,
	// and this subnet fits in the space,
	// it will be placed there.
	thirdNetwork, _ := service.CreateSubnet(ctx, net.CIDRMask(25, 32), "", nil)
	fmt.Println(thirdNetwork.String())

	// Request a fourth subnet.
	// As 2 /25s fit within a /24, both this subnet and the previous
	// subnet fit in the space of the older /24.
	fourthNetwork, _ := service.CreateSubnet(ctx, net.CIDRMask(25, 32), "", nil)
	fmt.Println(fourthNetwork.String())

	// Output:
	// 10.4.0.0/24
	// 10.4.1.0/32
	// 10.4.0.0/25
	// 10.4.0.128/25
}
