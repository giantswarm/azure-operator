package nodestatus

import (
	"context"

	"github.com/giantswarm/microerror"
	apicorev1 "k8s.io/api/core/v1"
	expcapzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
	"sigs.k8s.io/cluster-api/controllers/noderefutil"
	"sigs.k8s.io/cluster-api/util"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

type getNodeReferencesResult struct {
	references []apicorev1.ObjectReference
	available  int
	ready      int
}

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	machinePool, err := key.ToMachinePool(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	azureMachinePool := expcapzv1alpha3.AzureMachinePool{}
	err = r.ctrlClient.Get(ctx, ctrlclient.ObjectKey{Namespace: machinePool.Namespace, Name: machinePool.Spec.Template.Spec.InfrastructureRef.Name}, &azureMachinePool)
	if err != nil {
		return microerror.Mask(err)
	}

	cluster, err := util.GetClusterFromMetadata(ctx, r.ctrlClient, azureMachinePool.ObjectMeta)
	if err != nil {
		return microerror.Mask(err)
	}

	// Check that the MachinePool or AzureMachinePool haven't been deleted or in the process.
	if !machinePool.DeletionTimestamp.IsZero() || !azureMachinePool.DeletionTimestamp.IsZero() {
		return nil
	}

	machinePool.Status.InfrastructureReady = azureMachinePool.Status.Ready
	machinePool.Status.Replicas = azureMachinePool.Status.Replicas

	machinePool.Spec.ProviderIDList = azureMachinePool.Spec.ProviderIDList
	if len(azureMachinePool.Spec.ProviderIDList) == 0 {
		r.logger.LogCtx(ctx, "level", "debug", "message", "AzureMachinePool.Spec.ProviderIDList haven't been set yet")
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return nil
	}

	// Check that the Machine doesn't already have a NodeRefs.
	if machinePool.Status.Replicas == machinePool.Status.ReadyReplicas && len(machinePool.Status.NodeRefs) == int(machinePool.Status.ReadyReplicas) {
		return nil
	}

	tenantClusterK8sClient, err := r.getTenantClusterK8sClient(ctx, cluster)
	if err != nil {
		return microerror.Mask(err)
	}
	if err = r.deleteRetiredNodes(ctx, tenantClusterK8sClient.CtrlClient(), machinePool.Status.NodeRefs, machinePool.Spec.ProviderIDList); err != nil {
		return nil
	}

	nodeRefsResult, err := r.getNodeReferences(ctx, tenantClusterK8sClient.CtrlClient(), machinePool.Spec.ProviderIDList)
	if err != nil {
		if IsErrNoAvailableNodes(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "Cannot assign NodeRefs to MachinePool, no matching Nodes")
			return nil
		}

		return microerror.Mask(err)
	}

	machinePool.Status.ReadyReplicas = int32(nodeRefsResult.ready)
	machinePool.Status.AvailableReplicas = int32(nodeRefsResult.available)
	machinePool.Status.UnavailableReplicas = machinePool.Status.Replicas - machinePool.Status.AvailableReplicas
	machinePool.Status.NodeRefs = nodeRefsResult.references

	// First we update the spec field (that way `ProviderIDList` is updated) then the status field.
	// Making it the other way around would return early and never update the spec field.
	err = r.ctrlClient.Update(ctx, &machinePool)
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.ctrlClient.Status().Update(ctx, &machinePool)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "Set MachinePool's NodeRefs")

	return nil
}

// deleteRetiredNodes deletes nodes that don't have a corresponding ProviderID in Spec.ProviderIDList.
// A MachinePool infrastucture provider indicates an instance in the set has been deleted by
// removing its ProviderID from the slice.
func (r *Resource) deleteRetiredNodes(ctx context.Context, c ctrlclient.Client, nodeRefs []apicorev1.ObjectReference, providerIDList []string) error {
	nodeRefsMap := make(map[string]*apicorev1.Node, len(nodeRefs))
	for _, nodeRef := range nodeRefs {
		node := &apicorev1.Node{}
		if err := c.Get(ctx, ctrlclient.ObjectKey{Name: nodeRef.Name}, node); err != nil {
			r.logger.LogCtx(ctx, "level", "debug", "message", "Failed to get Node, skipping", "stack", microerror.JSON(microerror.Mask(err)))
			continue
		}

		nodeProviderID, err := noderefutil.NewProviderID(node.Spec.ProviderID)
		if err != nil {
			r.logger.LogCtx(ctx, "level", "debug", "message", "Failed to parse ProviderID, skipping", "providerID", node.Spec.ProviderID, "stack", microerror.JSON(microerror.Mask(err)))
			continue
		}

		nodeRefsMap[nodeProviderID.ID()] = node
	}
	for _, providerID := range providerIDList {
		pid, err := noderefutil.NewProviderID(providerID)
		if err != nil {
			r.logger.LogCtx(ctx, "level", "debug", "message", "Failed to parse ProviderID, skipping", "providerID", providerID, "stack", microerror.JSON(microerror.Mask(err)))
			continue
		}
		delete(nodeRefsMap, pid.ID())
	}
	for _, node := range nodeRefsMap {
		if err := c.Delete(ctx, node); err != nil {
			return microerror.Mask(err)
		}
	}
	return nil
}

// getNodeReferences will return an slice containing the k8s nodes whose Azure ID is inside the given providerIDList.
// This is useful when we want the nodes that belong to a certain MachinePool.
func (r *Resource) getNodeReferences(ctx context.Context, c ctrlclient.Client, providerIDList []string) (getNodeReferencesResult, error) {
	var ready, available int
	nodeRefsMap := make(map[string]apicorev1.Node)
	nodeList := apicorev1.NodeList{}
	for {
		if err := c.List(ctx, &nodeList, ctrlclient.Continue(nodeList.Continue)); err != nil {
			return getNodeReferencesResult{}, microerror.Mask(err)
		}

		for _, node := range nodeList.Items {
			nodeProviderID, err := noderefutil.NewProviderID(node.Spec.ProviderID)
			if err != nil {
				r.logger.LogCtx(ctx, "level", "debug", "message", "Failed to parse ProviderID, skipping", "providerID", node.Spec.ProviderID, "stack", microerror.JSON(microerror.Mask(err)))
				continue
			}

			nodeRefsMap[nodeProviderID.ID()] = node
		}

		if nodeList.Continue == "" {
			break
		}
	}

	var nodeRefs []apicorev1.ObjectReference
	for _, providerID := range providerIDList {
		pid, err := noderefutil.NewProviderID(providerID)
		if err != nil {
			r.logger.LogCtx(ctx, "level", "debug", "message", "Failed to parse ProviderID, skipping", "providerID", providerID, "stack", microerror.JSON(microerror.Mask(err)))
			continue
		}
		if node, ok := nodeRefsMap[pid.ID()]; ok {
			available++
			if nodeIsReady(&node) {
				ready++
			}
			nodeRefs = append(nodeRefs, apicorev1.ObjectReference{
				Kind:       node.Kind,
				APIVersion: node.APIVersion,
				Name:       node.Name,
				UID:        node.UID,
			})
		}
	}

	if len(nodeRefs) == 0 {
		return getNodeReferencesResult{}, errNoAvailableNodes
	}
	return getNodeReferencesResult{nodeRefs, available, ready}, nil
}

func nodeIsReady(node *apicorev1.Node) bool {
	for _, n := range node.Status.Conditions {
		if n.Type == apicorev1.NodeReady {
			return n.Status == apicorev1.ConditionTrue
		}
	}
	return false
}
