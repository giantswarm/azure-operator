package endpoints

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2017-09-01/network"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/client"
)

const (
	expands = ""
)

func (r *Resource) getMasterNICPrivateIP(resourceGroupName, networkInterfaceName string) (string, error) {
	interfacesClient, err := r.getInterfacesClient()
	if err != nil {
		return "", microerror.Mask(err)
	}

	networkInterface, err := interfacesClient.Get(
		context.Background(),
		resourceGroupName,
		networkInterfaceName,
		expands,
	)
	if err != nil {
		return "", microerror.Mask(err)
	}

	ipConfigurations := *networkInterface.IPConfigurations

	if len(ipConfigurations) != 1 {
		return "", microerror.Mask(incorrectNumberNetworkInterfacesError)
	}

	ipConfiguration := ipConfigurations[0]
	privateIP := *ipConfiguration.PrivateIPAddress

	if privateIP == "" {
		return "", microerror.Mask(privateIPAddressEmptyError)
	}

	return privateIP, nil
}

func (r *Resource) getInterfacesClient() (*network.InterfacesClient, error) {
	azureClients, err := client.NewAzureClientSet(r.azureConfig)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return azureClients.InterfacesClient, nil
}
