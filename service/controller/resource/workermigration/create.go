package workermigration

import (
	"context"

	"github.com/giantswarm/microerror"
	expcapiv1alpha3 "sigs.k8s.io/cluster-api/exp/api/v1alpha3"

	"github.com/giantswarm/azure-operator/v4/service/controller/key"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/workermigration/internal/azure"
)

// EnsureCreated ensures that built-in workers are migrated to node pool.
func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "ensuring that built-in workers are migrated to node pool")

	var builtinVMSS azure.VMSS
	{
		builtinVMSS, err = r.azureapi.GetVMSS(ctx, key.ResourceGroupName(cr), key.WorkerVMSSName(cr))
		if azure.IsNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "built-in workers don't exist anymore")
			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")

			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}
	}

	var machinePool expcapiv1alpha3.MachinePool
	//var azureMachinePool expcapzv1alpha3.AzureMachinePool
	{
		// TODO: Ensure that MachinePool & AzureMachinePool CRs exist for builtinVMSS.
	}

	{
		if !machinePool.Status.InfrastructureReady || (machinePool.Status.Replicas != machinePool.Status.ReadyReplicas) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "node pool workers are not ready yet")
			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")

			return nil
		}

		// TODO: Drain old workers.
	}

	{
		err = r.azureapi.DeleteVMSS(ctx, key.ResourceGroupName(cr), *builtinVMSS.Name)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", "built-in workers VMSS deleted")
	}

	return nil
}
