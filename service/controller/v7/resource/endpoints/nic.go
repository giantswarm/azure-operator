package endpoints

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-06-01/network"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/service/controller/v7/controllercontext"
)

const (
	expands = ""
)

func (r *Resource) getMasterNICPrivateIPs(ctx context.Context, resourceGroupName, virtualMachineScaleSetName string) ([]string, error) {
	var ips []string

	interfacesClient, err := r.getInterfacesClient(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	result, err := interfacesClient.ListVirtualMachineScaleSetNetworkInterfaces(
		context.Background(),
		resourceGroupName,
		virtualMachineScaleSetName,
	)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	for result.NotDone() {
		values := result.Values()
		for _, networkInterface := range values {
			ipConfigurations := *networkInterface.IPConfigurations
			if len(ipConfigurations) != 1 {
				return nil, microerror.Mask(incorrectNumberNetworkInterfacesError)
			}

			ipConfiguration := ipConfigurations[0]
			privateIP := *ipConfiguration.PrivateIPAddress
			if privateIP == "" {
				return nil, microerror.Mask(privateIPAddressEmptyError)
			}

			ips = append(ips, privateIP)
		}

		err := result.Next()
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return ips, nil
}

func (r *Resource) getInterfacesClient(ctx context.Context) (*network.InterfacesClient, error) {
	sc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return sc.AzureClientSet.InterfacesClient, nil
}
