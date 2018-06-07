package instance

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller/context/resourcecanceledcontext"

	"github.com/giantswarm/azure-operator/service/controller/v2/key"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	// Find the next instance ID and instance name we want to trigger the update
	// for. Instance names look something like the following example.
	//
	//     0gjpt-worker-000004
	//
	// Instance IDs are simple non negative integers.
	//
	//     4
	//
	var instanceID string
	var instanceName string
	{
		r.logger.LogCtx(ctx, "level", "debug", "message", "looking for the next instance to be updated")

		c, err := r.getVMsClient()
		if err != nil {
			return microerror.Mask(err)
		}

		g := key.ResourceGroupName(customObject)
		s := key.WorkerVMSSName(customObject)
		result, err := c.List(ctx, g, s, "", "", "")
		if IsScaleSetNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "did not find the scale set")
			resourcecanceledcontext.SetCanceled(ctx)
			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource for custom object")

			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}

		for _, v := range result.Values() {
			if *v.LatestModelApplied {
				continue
			}

			instanceID = *v.InstanceID
			instanceName = key.InstanceName(customObject, *v.InstanceID)

			if !key.IsFinalProvisioningState(*v.ProvisioningState) {
				r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("instance '%s' is in state '%s'", instanceName, *v.ProvisioningState))
				resourcecanceledcontext.SetCanceled(ctx)
				r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource for custom object")

				return nil
			}

			break
		}

		if instanceID == "" {
			r.logger.LogCtx(ctx, "level", "debug", "message", "no instance ID found that needs to be updated")
			resourcecanceledcontext.SetCanceled(ctx)
			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource for custom object")

			return nil
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found instance '%s' has to be updated", instanceName))
	}

	// Trigger the update for the found instance.
	{
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensuring instance '%s' to be updated", instanceName))

		c, err := r.getScaleSetsClient()
		if err != nil {
			return microerror.Mask(err)
		}

		g := key.ResourceGroupName(customObject)
		s := key.WorkerVMSSName(customObject)
		IDs := compute.VirtualMachineScaleSetVMInstanceRequiredIDs{
			InstanceIds: to.StringSlicePtr([]string{
				instanceID,
			}),
		}
		_, err = c.UpdateInstances(ctx, g, s, IDs)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensured instance '%s' to be updated", instanceName))
	}

	return nil
}
