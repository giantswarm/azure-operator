package setup

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-11-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/azure-operator/v4/e2e/env"
)

func bastion(ctx context.Context, config Config) error {
	var err error

	resourceGroupName := env.ClusterID()
	virtualNetworkName := fmt.Sprintf("%s-%s", resourceGroupName, "VirtualNetwork")
	masterSubnetName := fmt.Sprintf("%s-%s", virtualNetworkName, "MasterSubnet")
	location := env.AzureLocation()
	sshKeyData := env.SSHPublicKey()

	// Get subnet for master nodes. It could be the workers subnet as well. We will deploy the bastion in this subnet.
	var subnet network.Subnet
	{
		err = WaitForNetworkToBeCreated(ctx, config.Logger, resourceGroupName, virtualNetworkName, masterSubnetName, *config.AzureClient.SubnetsClient)
		if err != nil {
			return nil
		}

		subnet, err = config.AzureClient.SubnetsClient.Get(ctx, resourceGroupName, virtualNetworkName, masterSubnetName, "")
		if err != nil {
			return microerror.Mask(err)
		}
	}

	// Create public IPAddress for the bastion instance.
	var ip network.PublicIPAddress
	{
		ip, err = CreatePublicIP(ctx, location, resourceGroupName, *config.AzureClient.IPAddressesClient)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	// Create network interface for the VM.
	var nic network.Interface
	{
		nic, err = CreateNIC(ctx, location, resourceGroupName, subnet, ip, *config.AzureClient.InterfacesClient)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	// Create bastion virtual machine.
	{
		err = CreateVM(ctx, location, resourceGroupName, sshKeyData, nic, config)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	config.Logger.LogCtx(ctx, "message", "ensuring bastion", "sshCommand", fmt.Sprintf("ssh -A e2e@%s", *ip.IPAddress))

	return nil
}

func WaitForNetworkToBeCreated(ctx context.Context, logger micrologger.Logger, resourceGroupName, virtualNetworkName, masterSubnetName string, subnetsClient network.SubnetsClient) error {
	o := func() error {
		_, err := subnetsClient.Get(ctx, resourceGroupName, virtualNetworkName, masterSubnetName, "")
		if err != nil {
			return microerror.Mask(err)
		}

		return nil
	}
	n := backoff.NewNotifier(logger, ctx)
	b := backoff.NewConstant(backoff.LongMaxWait, backoff.LongMaxInterval)
	err := backoff.RetryNotify(o, b, n)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

// CreatePublicIP creates a new public IP.
func CreatePublicIP(ctx context.Context, location, groupName string, ipClient network.PublicIPAddressesClient) (ip network.PublicIPAddress, err error) {
	bastionE2EPublicIpName := "bastionE2EPublicIp"
	future, err := ipClient.CreateOrUpdate(
		ctx,
		groupName,
		bastionE2EPublicIpName,
		network.PublicIPAddress{
			Name:     to.StringPtr(bastionE2EPublicIpName),
			Location: to.StringPtr(location),
			Sku:      &network.PublicIPAddressSku{Name: network.PublicIPAddressSkuNameStandard},
			PublicIPAddressPropertiesFormat: &network.PublicIPAddressPropertiesFormat{
				PublicIPAddressVersion:   network.IPv4,
				PublicIPAllocationMethod: network.Static,
			},
		},
	)

	if err != nil {
		return ip, fmt.Errorf("cannot create public ip address: %v", err)
	}

	err = future.WaitForCompletionRef(ctx, ipClient.Client)
	if err != nil {
		return ip, fmt.Errorf("cannot get public ip address create or update future response: %v", err)
	}

	return future.Result(ipClient)
}

// CreateVM creates a new virtual machine with the specified name using the specified NIC.
func CreateVM(ctx context.Context, location, groupName, sshKeyData string, nic network.Interface, config Config) error {
	vmName := "bastionE2EVirtualMachine"
	username := "e2e"
	config.Logger.LogCtx(ctx, "message", "Creating e2e bastion instance", "vmName", vmName)

	_, err := config.AzureClient.VirtualMachinesClient.CreateOrUpdate(
		ctx,
		groupName,
		vmName,
		compute.VirtualMachine{
			Location: to.StringPtr(location),
			VirtualMachineProperties: &compute.VirtualMachineProperties{
				HardwareProfile: &compute.HardwareProfile{
					VMSize: compute.VirtualMachineSizeTypesBasicA0,
				},
				StorageProfile: &compute.StorageProfile{
					ImageReference: &compute.ImageReference{
						Publisher: to.StringPtr("CoreOS"),
						Offer:     to.StringPtr("CoreOS"),
						Sku:       to.StringPtr("Stable"),
						Version:   to.StringPtr("2135.6.0"),
					},
				},
				OsProfile: &compute.OSProfile{
					ComputerName:  to.StringPtr(vmName),
					AdminUsername: to.StringPtr(username),
					LinuxConfiguration: &compute.LinuxConfiguration{
						DisablePasswordAuthentication: to.BoolPtr(true),
						SSH: &compute.SSHConfiguration{
							PublicKeys: &[]compute.SSHPublicKey{
								{
									Path:    to.StringPtr(fmt.Sprintf("/home/%s/.ssh/authorized_keys", username)),
									KeyData: to.StringPtr(sshKeyData),
								},
							},
						},
					},
				},
				NetworkProfile: &compute.NetworkProfile{
					NetworkInterfaces: &[]compute.NetworkInterfaceReference{
						{
							ID: nic.ID,
							NetworkInterfaceReferenceProperties: &compute.NetworkInterfaceReferenceProperties{
								Primary: to.BoolPtr(true),
							},
						},
					},
				},
			},
		},
	)
	if err != nil {
		return fmt.Errorf("cannot create vm: %v", err)
	}

	return nil
}

// CreateNIC creates a new network interface in the passed subnet using the passed public ip address.
func CreateNIC(ctx context.Context, location, groupName string, subnet network.Subnet, ip network.PublicIPAddress, nicClient network.InterfacesClient) (nic network.Interface, err error) {
	nicName := "bastionE2ENIC"

	future, err := nicClient.CreateOrUpdate(ctx, groupName, nicName, network.Interface{
		Name:     to.StringPtr(nicName),
		Location: to.StringPtr(location),
		InterfacePropertiesFormat: &network.InterfacePropertiesFormat{
			IPConfigurations: &[]network.InterfaceIPConfiguration{
				{
					Name: to.StringPtr("ipConfig1"),
					InterfaceIPConfigurationPropertiesFormat: &network.InterfaceIPConfigurationPropertiesFormat{
						Subnet:                    &subnet,
						PrivateIPAllocationMethod: network.Dynamic,
						PublicIPAddress:           &ip,
					},
				},
			},
		},
	})
	if err != nil {
		return nic, fmt.Errorf("cannot create nic: %v", err)
	}

	err = future.WaitForCompletionRef(ctx, nicClient.Client)
	if err != nil {
		return nic, fmt.Errorf("cannot get nic create or update future response: %v", err)
	}

	return future.Result(nicClient)
}
