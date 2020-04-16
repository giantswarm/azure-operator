package instance

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/service/controller/key"
	"github.com/giantswarm/azure-operator/service/controller/resource/instance/internal/state"
)

func (r *Resource) terminateOldWorkersTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "finding all worker VMSS instances") // nolint: errcheck
	var allWorkerInstances []compute.VirtualMachineScaleSetVM
	{
		allWorkerInstances, err = r.allInstances(ctx, cr, key.WorkerVMSSName)
		if IsScaleSetNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("did not find the scale set '%s'", key.WorkerVMSSName(cr))) // nolint: errcheck
			r.logger.LogCtx(ctx, "level", "debug", "message", "restarting upgrade process")                                           // nolint: errcheck

			return DeploymentUninitialized, nil
		} else if err != nil {
			return DeploymentUninitialized, microerror.Mask(err)
		}
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found %d worker VMSS instances", len(allWorkerInstances))) // nolint: errcheck

	c, err := r.getScaleSetsClient(ctx)
	if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "filtering instance IDs for old instances") // nolint: errcheck

	g := key.ResourceGroupName(cr)
	s := key.WorkerVMSSName(cr)
	var ids compute.VirtualMachineScaleSetVMInstanceRequiredIDs
	{
		var strIds []string
		for _, i := range allWorkerInstances {
			old, err := r.isWorkerInstanceFromPreviousRelease(ctx, cr, i)
			if err != nil {
				return DeploymentUninitialized, nil
			}

			if old != nil && *old {
				strIds = append(strIds, *i.InstanceID)
			}
		}

		ids = compute.VirtualMachineScaleSetVMInstanceRequiredIDs{
			InstanceIds: to.StringSlicePtr(strIds),
		}
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "filtered instance IDs for old instances")                                 // nolint: errcheck
	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("terminating %d old worker instances", len(*ids.InstanceIds))) // nolint: errcheck

	res, err := c.DeleteInstances(ctx, g, s, ids)
	if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}
	_, err = c.DeleteInstancesResponder(res.Response())
	if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("terminated %d old worker instances", len(*ids.InstanceIds))) // nolint: errcheck

	return ScaleDownWorkerVMSS, nil
}
