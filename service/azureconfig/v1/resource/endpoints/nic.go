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

	privateIP := ""
	for _, ipConfiguration := range *networkInterface.IPConfigurations {
		if *ipConfiguration.PrivateIPAddress != "" {
			privateIP = *ipConfiguration.PrivateIPAddress
		}
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
