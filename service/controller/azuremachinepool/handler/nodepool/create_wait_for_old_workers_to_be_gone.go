package nodepool

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-12-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/giantswarm/microerror"
	"sigs.k8s.io/cluster-api/util"

	"github.com/giantswarm/azure-operator/v5/pkg/handler/nodes/state"
	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

func (r *Resource) waitForOldWorkersToBeGoneTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	azureMachinePool, err := key.ToAzureMachinePool(obj)
	if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}

	cluster, err := util.GetClusterFromMetadata(ctx, r.CtrlClient, azureMachinePool.ObjectMeta)
	if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}

	if !cluster.GetDeletionTimestamp().IsZero() {
		r.Logger.Debugf(ctx, "Cluster is being deleted, skipping reconciling node pool")
		return currentState, nil
	}

	oldInstances, _, err := r.splitInstancesByUpdatedStatus(ctx, azureMachinePool)
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	if len(oldInstances) > 0 {
		r.Logger.Debugf(ctx, "There are still %d workers from the previous release running", len(oldInstances))

		// We issue another "Delete" to the old instances because sometimes the "Simulate Eviction" API call we
		// use for spot instances node pools only stops the instances rather than deleting them.
		{
			var ids compute.VirtualMachineScaleSetVMInstanceRequiredIDs
			{
				var strIds []string
				for _, i := range oldInstances {
					strIds = append(strIds, *i.InstanceID)
				}

				ids = compute.VirtualMachineScaleSetVMInstanceRequiredIDs{
					InstanceIds: to.StringSlicePtr(strIds),
				}
			}

			virtualMachineScaleSetsClient, err := r.ClientFactory.GetVirtualMachineScaleSetsClient(ctx, azureMachinePool.ObjectMeta)
			if err != nil {
				return currentState, microerror.Mask(err)
			}

			res, err := virtualMachineScaleSetsClient.DeleteInstances(ctx, key.ClusterID(&azureMachinePool), key.NodePoolVMSSName(&azureMachinePool), ids, nil)
			if err != nil {
				return currentState, microerror.Mask(err)
			}
			_, err = virtualMachineScaleSetsClient.DeleteInstancesResponder(res.Response())
			if err != nil {
				return currentState, microerror.Mask(err)
			}
		}

		return currentState, nil
	}

	// Enable cluster autoscaler for this nodepool.
	err = r.enableClusterAutoscaler(ctx, azureMachinePool)
	if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}

	return DeploymentUninitialized, nil
}
