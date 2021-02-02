package nodepool

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	apiextensionslabels "github.com/giantswarm/apiextensions/v3/pkg/label"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

func (r *Resource) sortNodesByTenantVMState(ctx context.Context, tenantClusterK8sClient ctrlclient.Client, azureMachinePool *v1alpha3.AzureMachinePool, instanceNameFunc func(nodePoolId, instanceID string) string) ([]corev1.Node, []corev1.Node, error) {
	virtualMachineScaleSetsClient, err := r.ClientFactory.GetVirtualMachineScaleSetsClient(ctx, azureMachinePool.ObjectMeta)
	if err != nil {
		return nil, nil, microerror.Mask(err)
	}

	virtualMachineScaleSetVMsClient, err := r.ClientFactory.GetVirtualMachineScaleSetVMsClient(ctx, azureMachinePool.ObjectMeta)
	if err != nil {
		return nil, nil, microerror.Mask(err)
	}

	vmss, err := virtualMachineScaleSetsClient.Get(ctx, key.ClusterID(azureMachinePool), key.NodePoolVMSSName(azureMachinePool))
	if err != nil {
		return nil, nil, microerror.Mask(err)
	}

	var nodeList *corev1.NodeList
	{
		nodeList = &corev1.NodeList{}

		labelSelector := ctrlclient.MatchingLabels{apiextensionslabels.MachinePool: azureMachinePool.Name}
		err := tenantClusterK8sClient.List(ctx, nodeList, labelSelector)
		if err != nil {
			return nil, nil, microerror.Mask(err)
		}
	}

	var allWorkerInstances []compute.VirtualMachineScaleSetVM
	{
		r.Logger.Debugf(ctx, "finding all worker VMSS instances")

		allWorkerInstances, err = r.GetVMSSInstances(ctx, virtualMachineScaleSetVMsClient, key.ClusterID(azureMachinePool), key.NodePoolVMSSName(azureMachinePool))
		if err != nil {
			return nil, nil, microerror.Mask(err)
		}

		r.Logger.Debugf(ctx, "found %d worker VMSS instances", len(allWorkerInstances))
	}

	nodeMap := make(map[string]corev1.Node)
	for _, n := range nodeList.Items {
		nodeMap[n.GetName()] = n
	}

	var oldNodes []corev1.Node
	var newNodes []corev1.Node
	for _, i := range allWorkerInstances {
		name := instanceNameFunc(azureMachinePool.Name, *i.InstanceID)

		n, found := nodeMap[name]
		if !found {
			// When VMSS is scaling up there might be VM instances that haven't
			// registered as nodes in k8s yet. Hence not all instances are
			// found from node list.
			continue
		}

		outdated, err := r.isWorkerInstanceFromPreviousRelease(ctx, tenantClusterK8sClient, azureMachinePool.Name, i, vmss)
		if err != nil {
			return nil, nil, microerror.Mask(err)
		}
		if *outdated {
			oldNodes = append(oldNodes, n)
		} else {
			newNodes = append(newNodes, n)
		}
	}

	return oldNodes, newNodes, nil
}
