package nodepool

import (
	"context"
	"fmt"
	"strconv"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/giantswarm/microerror"
	capzexp "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1beta1"

	"github.com/giantswarm/azure-operator/v7/service/controller/key"
)

const (
	clusterAutoscalerEnabledTagName = "cluster-autoscaler-enabled"
)

func (r *Resource) disableClusterAutoscaler(ctx context.Context, azureMachinePool capzexp.AzureMachinePool) error {
	resourceGroup := key.ClusterID(&azureMachinePool)
	vmssName := key.NodePoolVMSSName(&azureMachinePool)

	r.Logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("Disabling cluster autoscaler for nodepool %s", vmssName))

	virtualMachineScaleSetsClient, err := r.ClientFactory.GetVirtualMachineScaleSetsClient(ctx, azureMachinePool.ObjectMeta)
	if err != nil {
		return microerror.Mask(err)
	}

	err = setClusterAutoscalerEnabled(ctx, virtualMachineScaleSetsClient, resourceGroup, vmssName, false)
	if err != nil {
		return microerror.Mask(err)
	}

	r.Logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("Disabled cluster autoscaler for nodepool %s", vmssName))

	return nil
}

func (r *Resource) enableClusterAutoscaler(ctx context.Context, azureMachinePool capzexp.AzureMachinePool) error {
	resourceGroup := key.ClusterID(&azureMachinePool)
	vmssName := key.NodePoolVMSSName(&azureMachinePool)

	r.Logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("Enabling cluster autoscaler for nodepool %s", vmssName))

	virtualMachineScaleSetsClient, err := r.ClientFactory.GetVirtualMachineScaleSetsClient(ctx, azureMachinePool.ObjectMeta)
	if err != nil {
		return microerror.Mask(err)
	}

	err = setClusterAutoscalerEnabled(ctx, virtualMachineScaleSetsClient, resourceGroup, vmssName, true)
	if err != nil {
		return microerror.Mask(err)
	}

	r.Logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("Enabled cluster autoscaler for nodepool %s", vmssName))

	return nil
}

func setClusterAutoscalerEnabled(ctx context.Context, virtualMachineScaleSetsClient *compute.VirtualMachineScaleSetsClient, resourceGroup, vmssName string, enabled bool) error {
	vmss, err := virtualMachineScaleSetsClient.Get(ctx, resourceGroup, vmssName)
	if err != nil {
		return microerror.Mask(err)
	}

	tags := vmss.Tags
	tags[clusterAutoscalerEnabledTagName] = to.StringPtr(strconv.FormatBool(enabled))

	params := compute.VirtualMachineScaleSetUpdate{
		Tags: tags,
	}

	_, err = virtualMachineScaleSetsClient.Update(ctx, resourceGroup, vmssName, params)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
