package nodepool

import (
	"context"
	"fmt"
	"strconv"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/giantswarm/microerror"
)

const (
	clusterAutoscalerEnabledTagName = "cluster-autoscaler-enabled"
)

func (r *Resource) disableClusterAutoscaler(ctx context.Context, virtualMachineScaleSetsClient *compute.VirtualMachineScaleSetsClient, resourceGroup, vmssName string) error {
	r.Logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("Disabling cluster autoscaler for nodepool %s", vmssName))

	err := setClusterAutoscalerEnabled(ctx, virtualMachineScaleSetsClient, resourceGroup, vmssName, false)
	if err != nil {
		return microerror.Mask(err)
	}

	r.Logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("Disabled cluster autoscaler for nodepool %s", vmssName))

	return nil
}

func (r *Resource) enableClusterAutoscaler(ctx context.Context, virtualMachineScaleSetsClient *compute.VirtualMachineScaleSetsClient, resourceGroup, vmssName string) error {
	r.Logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("Enabling cluster autoscaler for nodepool %s", vmssName))

	err := setClusterAutoscalerEnabled(ctx, virtualMachineScaleSetsClient, resourceGroup, vmssName, true)
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
