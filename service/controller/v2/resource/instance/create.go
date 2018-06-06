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

	// Find the next instance ID we want to trigger the update for. Instance IDs
	// look something like the following example. Anyways, the instance ID the
	// Azure API expects when triggering updates is a simple non negative integer.
	// The computation of the final instance ID is done by the resource manager
	// internally.
	//
	//     0gjpt-worker-000004
	//
	var instanceID string
	{
		r.logger.LogCtx(ctx, "level", "debug", "message", "looking for the next instance to be updated")

		c, err := r.getVMsClient()
		if err != nil {
			return microerror.Mask(err)
		}

		g := key.ResourceGroupName(customObject)
		s := key.WorkerVMSSName(customObject)
		result, err := c.List(ctx, g, s, "", "", "")
		if err != nil {
			return microerror.Mask(err)
		}

		for _, v := range result.Values() {
			if *v.LatestModelApplied {
				continue
			}

			instanceID = *v.InstanceID

			if !key.IsFinalProvisioningState(*v.ProvisioningState) {
				r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("instance '%s' is in state '%s'", instanceID, *v.ProvisioningState))
				resourcecanceledcontext.SetCanceled(ctx)
				r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource for custom object")

				return nil
			}

			break
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found instance '%s' has to be updated", instanceID))
	}

	// Trigger the update for the found instance.
	{
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensuring instance '%s' to be updated", instanceID))

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

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensured instance '%s' to be updated", instanceID))
	}

	return nil
}
