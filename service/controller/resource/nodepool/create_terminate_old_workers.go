package nodepool

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/giantswarm/errors/tenant"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/tenantcluster/v3/pkg/tenantcluster"
	"sigs.k8s.io/cluster-api/util"

	"github.com/giantswarm/azure-operator/v4/service/controller/internal/state"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

func (r *Resource) terminateOldWorkersTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	azureMachinePool, err := key.ToAzureMachinePool(obj)
	if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}

	cluster, err := util.GetClusterFromMetadata(ctx, r.CtrlClient, azureMachinePool.ObjectMeta)
	if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}

	virtualMachineScaleSetVMsClient, err := r.ClientFactory.GetVirtualMachineScaleSetVMsClient(ctx, azureMachinePool.ObjectMeta)
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	virtualMachineScaleSetsClient, err := r.ClientFactory.GetVirtualMachineScaleSetsClient(ctx, azureMachinePool.ObjectMeta)
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	tenantClusterK8sClient, err := r.getTenantClusterK8sClient(ctx, cluster)
	if tenantcluster.IsTimeout(err) {
		r.Logger.LogCtx(ctx, "level", "debug", "message", "timeout fetching certificates")
		r.Logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return currentState, nil
	} else if tenant.IsAPINotAvailable(err) {
		r.Logger.LogCtx(ctx, "level", "debug", "message", "tenant API not available yet")
		r.Logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return currentState, nil
	} else if err != nil {
		return currentState, microerror.Mask(err)
	}

	var allWorkerInstances []compute.VirtualMachineScaleSetVM
	{
		r.Logger.LogCtx(ctx, "level", "debug", "message", "finding all worker VMSS instances")

		allWorkerInstances, err = r.GetVMSSInstances(ctx, virtualMachineScaleSetVMsClient, key.ClusterID(&azureMachinePool), key.NodePoolVMSSName(&azureMachinePool))
		if err != nil {
			return DeploymentUninitialized, microerror.Mask(err)
		}

		r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found %d worker VMSS instances", len(allWorkerInstances)))
	}

	resourceGroupName := key.ClusterID(&azureMachinePool)
	nodePoolVMSSName := key.NodePoolVMSSName(&azureMachinePool)
	vmss, err := virtualMachineScaleSetsClient.Get(ctx, resourceGroupName, nodePoolVMSSName)
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	r.Logger.LogCtx(ctx, "level", "debug", "message", "filtering instance IDs for old instances")

	var ids compute.VirtualMachineScaleSetVMInstanceRequiredIDs
	{
		var strIds []string
		for _, i := range allWorkerInstances {
			old, err := r.isWorkerInstanceFromPreviousRelease(ctx, tenantClusterK8sClient, azureMachinePool.Name, i, vmss)
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

	r.Logger.LogCtx(ctx, "level", "debug", "message", "filtered instance IDs for old instances")
	r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("terminating %d old worker instances", len(*ids.InstanceIds)))

	res, err := virtualMachineScaleSetsClient.DeleteInstances(ctx, resourceGroupName, nodePoolVMSSName, ids)
	if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}
	_, err = virtualMachineScaleSetsClient.DeleteInstancesResponder(res.Response())
	if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}

	r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("terminated %d old worker instances", len(*ids.InstanceIds)))

	return ScaleDownWorkerVMSS, nil
}
