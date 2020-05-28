package nodes

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v4/service/controller/controllercontext"
)

func (r *Resource) GetStorageAccountsClient(ctx context.Context) (*storage.AccountsClient, error) {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return cc.AzureClientSet.StorageAccountsClient, nil
}

func (r *Resource) GetVMsClient(ctx context.Context) (*compute.VirtualMachineScaleSetVMsClient, error) {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return cc.AzureClientSet.VirtualMachineScaleSetVMsClient, nil
}
