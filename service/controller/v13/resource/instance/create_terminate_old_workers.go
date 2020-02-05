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

	r.logger.LogCtx(ctx, "level", "debug", "message", "finding all worker VMSS instances")
	var allWorkerInstances []compute.VirtualMachineScaleSetVM
	{
		allWorkerInstances, err = r.allInstances(ctx, customObject, key.WorkerVMSSName)
		if IsScaleSetNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("did not find the scale set '%s'", key.WorkerVMSSName(customObject)))
			r.logger.LogCtx(ctx, "level", "debug", "message", "restarting upgrade process")

			return DeploymentUninitialized, nil
		} else if err != nil {
			return "", microerror.Mask(err)
		}
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found %d worker VMSS instances", len(allWorkerInstances)))

	c, err := r.getScaleSetsClient(ctx)
	if err != nil {
		return "", microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "filtering instance IDs for old instances")

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

	r.logger.LogCtx(ctx, "level", "debug", "message", "filtered instance IDs for old instances")
	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("terminating %d old worker instances", len(*ids.InstanceIds)))

	res, err := c.DeleteInstances(ctx, g, s, ids)
	if err != nil {
		return "", microerror.Mask(err)
	}
	_, err = c.DeleteInstancesResponder(res.Response())
	if err != nil {
		return "", microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("terminated %d old worker instances", len(*ids.InstanceIds)))

	return ScaleDownWorkerVMSS, nil
}
