package setup

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-06-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/integration/env"
)

func bastion(ctx context.Context, config Config) error {
	var err error

	resourceGroupName := env.ClusterID()
	virtualNetworkName := fmt.Sprintf("%s-%s", resourceGroupName, "VirtualNetwork")
	masterSecurityGroupName := fmt.Sprintf("%s-%s", resourceGroupName, "MasterSecurityGroup")
	workerSecurityGroupName := fmt.Sprintf("%s-%s", resourceGroupName, "WorkerSecurityGroup")
	subnetAddressPrefix := env.BastionE2ESubnetCIDR()
	location := env.AzureLocation()

	// Create the bastion security group allowing SSH from everywhere
	var sg network.SecurityGroup
	{
		sg, err = CreateNetworkSecurityGroup(ctx, location, resourceGroupName, config)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	// Add subnet to virtual network, attach the bastion security group
	var subnet network.Subnet
	{
		subnet, err = CreateVirtualNetworkSubnet(ctx, resourceGroupName, virtualNetworkName, subnetAddressPrefix, sg, config)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	// Change workers/master security group to allow traffic from created subnet
	_, err = CreateSSHRule(ctx, resourceGroupName, masterSecurityGroupName, subnetAddressPrefix, *config.AzureClient.SecurityRulesClient)
	if err != nil {
		return microerror.Mask(err)
	}
	_, err = CreateSSHRule(ctx, resourceGroupName, workerSecurityGroupName, subnetAddressPrefix, *config.AzureClient.SecurityRulesClient)
	if err != nil {
		return microerror.Mask(err)
	}

	// Create public IPAddress for the instance
	var ip network.PublicIPAddress
	{
		ip, err = CreatePublicIP(ctx, location, resourceGroupName, *config.AzureClient.IPAddressesClient)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	// Create network interface for the VM
	var nic network.Interface
	{
		nic, err = CreateNIC(ctx, location, resourceGroupName, subnet, ip, *config.AzureClient.InterfacesClient)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	// Create VM in that subnet, using our SSH keys
	{
		_, err = CreateVM(ctx, location, resourceGroupName, nic, config)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

// CreateVirtualNetworkSubnet creates a subnet in an existing vnet
func CreateVirtualNetworkSubnet(ctx context.Context, groupName string, vnetName string, subnetAddressPrefix string, securityGroup network.SecurityGroup, config Config) (subnet network.Subnet, err error) {
	bastionE2ESubnetName := "bastionE2ESubnet"
	config.Logger.LogCtx(ctx, "message", "Adding e2e bastion subnet to virtual network", "subnetName", bastionE2ESubnetName, "cidr", subnetAddressPrefix)
	future, err := config.AzureClient.SubnetsClient.CreateOrUpdate(
		ctx,
		groupName,
		vnetName,
		bastionE2ESubnetName,
		network.Subnet{
			SubnetPropertiesFormat: &network.SubnetPropertiesFormat{
				AddressPrefix:        to.StringPtr(subnetAddressPrefix),
				NetworkSecurityGroup: &securityGroup,
				RouteTable:           nil,
			},
		})
	if err != nil {
		return subnet, fmt.Errorf("cannot create subnet: %v", err)
	}

	err = future.WaitForCompletionRef(ctx, config.AzureClient.SubnetsClient.Client)
	if err != nil {
		return subnet, fmt.Errorf("cannot get the subnet create or update future response: %v", err)
	}

	return future.Result(*config.AzureClient.SubnetsClient)
}

// CreateSSHRule creates an inbound network security rule that allows using port 22
func CreateSSHRule(ctx context.Context, groupName, nsgName, subnetAddressPrefix string, rulesClient network.SecurityRulesClient) (rule network.SecurityRule, err error) {
	future, err := rulesClient.CreateOrUpdate(ctx,
		groupName,
		nsgName,
		"ALLOW-SSH-FROM-BASTION-E2E",
		network.SecurityRule{
			SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
				Access:                   network.SecurityRuleAccessAllow,
				DestinationAddressPrefix: to.StringPtr("*"),
				DestinationPortRange:     to.StringPtr("22"),
				Direction:                network.SecurityRuleDirectionInbound,
				Description:              to.StringPtr("Allow SSH from bastion e2e subnet"),
				Priority:                 to.Int32Ptr(103),
				Protocol:                 network.SecurityRuleProtocolTCP,
				SourceAddressPrefix:      to.StringPtr(subnetAddressPrefix),
				SourcePortRange:          to.StringPtr("*"),
			},
		})
	if err != nil {
		return rule, fmt.Errorf("cannot create SSH security rule: %v", err)
	}

	err = future.WaitForCompletionRef(ctx, rulesClient.Client)
	if err != nil {
		return rule, fmt.Errorf("cannot get security rule create or update future response: %v", err)
	}

	return future.Result(rulesClient)
}

// CreateNetworkSecurityGroup creates a new network security group with rules set for allowing SSH and HTTPS use
func CreateNetworkSecurityGroup(ctx context.Context, location, groupName string, config Config) (nsg network.SecurityGroup, err error) {
	securityGroupName := "bastionE2ESecurityGroup"

	config.Logger.LogCtx(ctx, "message", "Creating e2e bastion security group", "securityGroup", securityGroupName)
	future, err := config.AzureClient.SecurityGroupsClient.CreateOrUpdate(
		ctx,
		groupName,
		securityGroupName,
		network.SecurityGroup{
			Location: to.StringPtr(location),
			SecurityGroupPropertiesFormat: &network.SecurityGroupPropertiesFormat{
				SecurityRules: &[]network.SecurityRule{
					{
						Name: to.StringPtr("allow_ssh_from_everywhere"),
						SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
							Protocol:                 network.SecurityRuleProtocolTCP,
							SourceAddressPrefix:      to.StringPtr("0.0.0.0/0"),
							SourcePortRange:          to.StringPtr("*"),
							DestinationAddressPrefix: to.StringPtr("0.0.0.0/0"),
							DestinationPortRange:     to.StringPtr("22"),
							Access:                   network.SecurityRuleAccessAllow,
							Direction:                network.SecurityRuleDirectionInbound,
							Priority:                 to.Int32Ptr(100),
						},
					},
				},
			},
		},
	)

	if err != nil {
		return nsg, fmt.Errorf("cannot create nsg: %v", err)
	}

	err = future.WaitForCompletionRef(ctx, config.AzureClient.SecurityGroupsClient.Client)
	if err != nil {
		return nsg, fmt.Errorf("cannot get nsg create or update future response: %v", err)
	}

	return future.Result(*config.AzureClient.SecurityGroupsClient)
}

// CreatePublicIP creates a new public IP
func CreatePublicIP(ctx context.Context, location, groupName string, ipClient network.PublicIPAddressesClient) (ip network.PublicIPAddress, err error) {
	bastionE2EPublicIpName := "bastionE2EPublicIp"
	future, err := ipClient.CreateOrUpdate(
		ctx,
		groupName,
		bastionE2EPublicIpName,
		network.PublicIPAddress{
			Name:     to.StringPtr(bastionE2EPublicIpName),
			Location: to.StringPtr(location),
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
// Username, password, and sshPublicKeyPath determine logon credentials.
func CreateVM(ctx context.Context, location, groupName string, nic network.Interface, config Config) (vm compute.VirtualMachine, err error) {
	vmName := "bastionE2EVirtualMachine"
	username := "e2e"
	config.Logger.LogCtx(ctx, "message", "Creating e2e bastion instance", "vmName", vmName)

	sshKeyData := "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDBSSJCLkZWhOvs6blotU+fWbrTmC7fOwOm0+w01Ww/YN3j3j1vCrvji1A4Yonr89ePQEQKfZsYcYFodQI/D3Uzu9rOFy0dCMQfvL/J6N8LkNtmooh3J2p061829MurAdD+TVsNGrD2FZGm5Ab4NiyDXIGAYCaHL6BHP16ipBglYjLQt6jVyzdTbYspkRi1QrsNFN3gIv9V47qQSvoNEsC97gvumKzCSQ/EwJzFoIlqVkZZHZTXvGwnZrAVXB69t9Y8OJ5zA6cYFAKR0O7lEiMpebdLNGkZgMA6t2PADxfT78PHkYXLR/4tchVuOSopssJqgSs7JgIktEE14xKyNyoLKIyBBo3xwywnDySsL8R2zG4Ytw1luo79pnSpIzTvfwrNhd7Cg//OYzyDCty+XUEUQx2JfOBx5Qb1OFw71WA+zYqjbworOsy2ZZ9UAy8ryjiaeT8L2ZRGuhdicD6kkL3Lxg5UeNIxS2FLNwgepZ4D8Vo6Yxe+VOZl524ffoOJSHQ0Gz8uE76hXMNEcn4t8HVkbR4sCMgLn2YbwJ2dJcROj4w80O4qgtN1vsL16r4gt9o6euml8LbmnJz6MtGdMczSO7kHRxirtEHMTtYbT1wNgUAzimbScRggBpUz5gbz+NRE1Xgnf4A5yNMRy+JOWtLVUozJlcGSiQkVcexzdb27yQ=="

	future, err := config.AzureClient.VirtualMachinesClient.CreateOrUpdate(
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
						SSH: &compute.SSHConfiguration{
							PublicKeys: &[]compute.SSHPublicKey{
								{
									Path: to.StringPtr(
										fmt.Sprintf("/home/%s/.ssh/authorized_keys",
											username)),
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
		return vm, fmt.Errorf("cannot create vm: %v", err)
	}

	err = future.WaitForCompletionRef(ctx, config.AzureClient.VirtualMachinesClient.Client)
	if err != nil {
		return vm, fmt.Errorf("cannot get the vm create or update future response: %v", err)
	}

	return future.Result(*config.AzureClient.VirtualMachinesClient)
}

// CreateNIC creates a new network interface. The Network Security Group is not a required parameter
func CreateNIC(ctx context.Context, location, groupName string, subnet network.Subnet, ip network.PublicIPAddress, nicClient network.InterfacesClient) (nic network.Interface, err error) {
	nicName := "bastionE2ENIC"
	nicParams := network.Interface{
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
	}

	//if nsgName != "" {
	//	nsg, err := GetNetworkSecurityGroup(ctx, nsgName)
	//	if err != nil {
	//		log.Fatalf("failed to get nsg: %v", err)
	//	}
	//	nicParams.NetworkSecurityGroup = &nsg
	//}

	future, err := nicClient.CreateOrUpdate(ctx, groupName, nicName, nicParams)
	if err != nil {
		return nic, fmt.Errorf("cannot create nic: %v", err)
	}

	err = future.WaitForCompletionRef(ctx, nicClient.Client)
	if err != nil {
		return nic, fmt.Errorf("cannot get nic create or update future response: %v", err)
	}

	return future.Result(nicClient)
}
