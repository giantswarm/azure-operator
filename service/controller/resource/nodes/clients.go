package nodes

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v4/service/controller/controllercontext"
)

func (r *Resource) GetVMsClient(ctx context.Context) (*compute.VirtualMachineScaleSetVMsClient, error) {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return cc.AzureClientSet.VirtualMachineScaleSetVMsClient, nil
}
