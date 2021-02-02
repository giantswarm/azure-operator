package nodestatus

import (
	"context"

	"github.com/giantswarm/apiextensions/v3/pkg/label"
	"github.com/giantswarm/microerror"
	apicorev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	expcapzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
	"sigs.k8s.io/cluster-api/controllers/noderefutil"
	"sigs.k8s.io/cluster-api/util"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v5/pkg/tenantcluster"
	"github.com/giantswarm/azure-operator/v5/service/controller/key"
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
		r.logger.Debugf(ctx, "object is being deleted")
		r.logger.Debugf(ctx, "canceling resource")
		return nil
	}

	machinePool.Spec.ProviderIDList = azureMachinePool.Spec.ProviderIDList
	if len(azureMachinePool.Spec.ProviderIDList) == 0 {
		r.logger.Debugf(ctx, "AzureMachinePool.Spec.ProviderIDList haven't been set yet")
		r.logger.Debugf(ctx, "canceling resource")
		return nil
	}

	err = r.ctrlClient.Update(ctx, &machinePool)
	if apierrors.IsConflict(err) {
		r.logger.Debugf(ctx, "conflict trying to save object in k8s API concurrently")
		r.logger.Debugf(ctx, "cancelling resource")
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	err = r.ctrlClient.Get(ctx, ctrlclient.ObjectKey{Name: machinePool.Name, Namespace: machinePool.Namespace}, &machinePool)
	if err != nil {
		return microerror.Mask(err)
	}

	tenantClusterK8sClient, err := r.tenantClientFactory.GetClient(ctx, cluster)
	if tenantcluster.IsAPINotAvailableError(err) {
		r.logger.Debugf(ctx, "tenant API not available yet")
		r.logger.Debugf(ctx, "canceling resource")

		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	nodeRefsResult, err := r.getNodeReferences(ctx, tenantClusterK8sClient, machinePool.Name, machinePool.Spec.ProviderIDList)
	if err != nil {
		if IsErrNoAvailableNodes(err) {
			r.logger.Debugf(ctx, "Cannot assign NodeRefs to MachinePool, no matching Nodes")
			r.logger.Debugf(ctx, "canceling resource")
			return nil
		}

		return microerror.Mask(err)
	}

	machinePool.Status.Replicas = azureMachinePool.Status.Replicas
	machinePool.Status.ReadyReplicas = int32(nodeRefsResult.ready)
	machinePool.Status.AvailableReplicas = int32(nodeRefsResult.available)
	machinePool.Status.UnavailableReplicas = machinePool.Status.Replicas - machinePool.Status.AvailableReplicas
	machinePool.Status.NodeRefs = nodeRefsResult.references

	err = r.ctrlClient.Status().Update(ctx, &machinePool)
	if apierrors.IsConflict(err) {
		r.logger.Debugf(ctx, "conflict trying to save object in k8s API concurrently")
		r.logger.Debugf(ctx, "cancelling resource")
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "Set MachinePool's NodeRefs")

	return nil
}

// getNodeReferences will return an slice containing the k8s nodes whose Azure ID is inside the given providerIDList.
// This is useful when we want the nodes that belong to a certain MachinePool.
func (r *Resource) getNodeReferences(ctx context.Context, c ctrlclient.Client, machinePoolName string, providerIDList []string) (getNodeReferencesResult, error) {
	var ready, available int
	nodeRefsMap := make(map[string]apicorev1.Node)
	nodeList := apicorev1.NodeList{}
	for {
		if err := c.List(ctx, &nodeList, ctrlclient.MatchingLabels{label.MachinePool: machinePoolName}, ctrlclient.Continue(nodeList.Continue)); err != nil {
			return getNodeReferencesResult{}, microerror.Mask(err)
		}

		for _, node := range nodeList.Items {
			nodeProviderID, err := noderefutil.NewProviderID(node.Spec.ProviderID)
			if err != nil {
				r.logger.Debugf(ctx, "Failed to parse ProviderID, skipping", "providerID", node.Spec.ProviderID)
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
			r.logger.Debugf(ctx, "Failed to parse ProviderID, skipping", "providerID", providerID)
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
