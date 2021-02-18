package endpoints

import (
	"context"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

func (r *Resource) getMasterNICPrivateIPs(ctx context.Context, cr v1alpha1.AzureConfig) ([]string, error) {
	var ips []string
	resourceGroupName := key.ClusterID(&cr)
	virtualMachineScaleSetName := key.MasterVMSSName(cr)

	interfacesClient, err := r.wcAzureClientFactory.GetInterfacesClient(ctx, key.ClusterID(&cr))
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
