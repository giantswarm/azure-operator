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

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("EnsureCreated called for cluster ID '%s'", key.ClusterID(customObject)))

	// Find the next instance ID we want to trigger the update for. Instance IDs
	// look something like the following example.
	//
	//     0gjpt-worker-000004
	//
	var instanceID string
	{
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

			instanceID = fmt.Sprintf("%s-worker-%06s\n", key.ClusterID(customObject), *v.InstanceID)

			if !key.IsFinalProvisioningState(*v.ProvisioningState) {
				r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("instance '%s' is in state '%s'", instanceID, *v.ProvisioningState))
				resourcecanceledcontext.SetCanceled(ctx)
				r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource for custom object")
			}

			break
		}
	}

	// TODO trigger update for found instance
	{
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
	}

	return nil
}
