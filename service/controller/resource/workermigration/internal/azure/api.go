package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	providerv1alpha1 "github.com/giantswarm/apiextensions/v2/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v4/client"
)

type api struct {
	clientFactory *client.Factory
	credentials   *providerv1alpha1.CredentialSecret
}

func GetAPI(f *client.Factory, credentials *providerv1alpha1.CredentialSecret) API {
	return &api{
		clientFactory: f,
		credentials:   credentials,
	}
}

func (a *api) GetVMSS(ctx context.Context, resourceGroupName, vmssName string) (VMSS, error) {
	client, err := a.clientFactory.GetVirtualMachineScaleSetsClient(a.credentials.Namespace, a.credentials.Name)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	vmss, err := client.Get(ctx, resourceGroupName, vmssName)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return &vmss, nil
}

func (a *api) DeleteVMSS(ctx context.Context, resourceGroupName, vmssName string) error {
	client, err := a.clientFactory.GetVirtualMachineScaleSetsClient(a.credentials.Namespace, a.credentials.Name)
	if err != nil {
		return microerror.Mask(err)
	}

	_, err = client.Delete(ctx, resourceGroupName, vmssName)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (a *api) ListVMSSNodes(ctx context.Context, resourceGroupName, vmssName string) (VMSSNodes, error) {
	client, err := a.clientFactory.GetVirtualMachineScaleSetVMsClient(a.credentials.Namespace, a.credentials.Name)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	result, err := client.List(ctx, resourceGroupName, vmssName, "", "", "")
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var vms []compute.VirtualMachineScaleSetVM
	for result.NotDone() {
		vms = append(vms, result.Values()...)

		err = result.NextWithContext(ctx)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return vms, nil
}
