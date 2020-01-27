package instance

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/giantswarm/azure-operator/service/controller/v13/key"
	"github.com/giantswarm/azure-operator/service/controller/v13/resource/instance/internal/state"
	"github.com/giantswarm/microerror"
)

func (r *Resource) terminateOldWorkersTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return "", microerror.Mask(err)
	}

	var allWorkerInstances []compute.VirtualMachineScaleSetVM
	{
		allWorkerInstances, err = r.allInstances(ctx, customObject, key.WorkerVMSSName)
		if IsScaleSetNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("did not find the scale set '%s'", key.WorkerVMSSName(customObject)))

			return currentState, nil
		} else if err != nil {
			return "", microerror.Mask(err)
		}
	}

	c, err := r.getScaleSetsClient(ctx)
	if err != nil {
		return "", microerror.Mask(err)
	}

	g := key.ResourceGroupName(customObject)
	s := key.WorkerVMSSName(customObject)
	var ids compute.VirtualMachineScaleSetVMInstanceRequiredIDs
	{
		var strIds []string
		for _, i := range allWorkerInstances {
			if !*i.LatestModelApplied {
				strIds = append(strIds, *i.InstanceID)
			}
		}

		ids = compute.VirtualMachineScaleSetVMInstanceRequiredIDs{
			InstanceIds: to.StringSlicePtr(strIds),
		}
	}

	res, err := c.DeleteInstances(ctx, g, s, ids)
	if err != nil {
		return "", microerror.Mask(err)
	}
	_, err = c.DeleteInstancesResponder(res.Response())
	if err != nil {
		return "", microerror.Mask(err)
	}

	return ScaleDownWorkerVMSS, nil
}
