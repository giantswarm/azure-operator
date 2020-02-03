package instance

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/giantswarm/azure-operator/service/controller/v13/controllercontext"
	"github.com/giantswarm/azure-operator/service/controller/v13/key"
	"github.com/giantswarm/azure-operator/service/controller/v13/resource/instance/internal/state"
)

const (
	// UnschedulablePatch is the JSON patch structure being applied to nodes using
	// a strategic merge patch in order to cordon them.
	UnschedulablePatch = `{"spec":{"unschedulable":true}}`
)

func (r *Resource) cordonOldWorkersTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return "", microerror.Mask(err)
	}
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return "", microerror.Mask(err)
	}

	if cc.Client.TenantCluster.K8s == nil {
		r.logger.LogCtx(ctx, "level", "debug", "message", "tenant cluster client not available yet")
		return currentState, nil
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "finding all worker VMSS instances")

	var allWorkerInstances []compute.VirtualMachineScaleSetVM
	{
		allWorkerInstances, err = r.allInstances(ctx, customObject, key.WorkerVMSSName)
		if IsScaleSetNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("did not find the scale set '%s'", key.WorkerVMSSName(customObject)))

			return currentState, nil
		} else if err != nil {
			return "", microerror.Mask(err)
		}
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found %d worker VMSS instances", len(allWorkerInstances)))
	r.logger.LogCtx(ctx, "level", "debug", "message", "finding all tenant cluster nodes")

	var nodes []corev1.Node
	{
		nodeList, err := cc.Client.TenantCluster.K8s.CoreV1().Nodes().List(metav1.ListOptions{})
		if err != nil {
			return "", microerror.Mask(err)
		}
		nodes = nodeList.Items
	}

	oldNodes, newNodes := sortNodesByTenantVMState(nodes, allWorkerInstances, customObject, key.WorkerInstanceName)
	if len(newNodes) < len(oldNodes) {
		// Wait until there's enough new nodes up.
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("number of new nodes (%d) is smaller than number of old nodes (%d)", len(newNodes), len(oldNodes)))
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return currentState, nil
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found %d old and %d new nodes from tenant cluster", len(oldNodes), len(newNodes)))
	r.logger.LogCtx(ctx, "level", "debug", "message", "ensuring old nodes are cordoned")

	oldNodesCordoned, err := r.ensureNodesCordoned(ctx, oldNodes)
	if err != nil {
		return "", microerror.Mask(err)
	}

	if oldNodesCordoned < len(oldNodes) {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("not all old nodes are still cordoned; %d pending", len(oldNodes)-oldNodesCordoned))

		return currentState, nil
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensured all old nodes (%d) are cordoned", oldNodesCordoned))

	return WaitForWorkersToBecomeReady, nil
}

// ensureNodesCordoned ensures that given tenant cluster nodes are cordoned.
func (r *Resource) ensureNodesCordoned(ctx context.Context, nodes []corev1.Node) (int, error) {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return 0, microerror.Mask(err)
	}

	var count int
	for _, n := range nodes {
		// Node already cordoned?
		if n.Spec.Unschedulable {
			count++
			continue
		}

		t := types.StrategicMergePatchType
		p := []byte(UnschedulablePatch)

		_, err = cc.Client.TenantCluster.K8s.CoreV1().Nodes().Patch(n.Name, t, p)
		if apierrors.IsNotFound(err) {
			// It might happen the node we want to drain got already removed. This
			// might even be due to human intervention. In case we cannot find the
			// node we assume the draining was successful and set the drainer config
			// status accordingly.

			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("tenant cluster node %s not found", n.Name))
		} else if err != nil {
			return 0, microerror.Mask(err)
		}

		count++
	}

	return count, nil
}

func sortNodesByTenantVMState(nodes []corev1.Node, instances []compute.VirtualMachineScaleSetVM, customObject providerv1alpha1.AzureConfig, instanceNameFunc func(customObject providerv1alpha1.AzureConfig, instanceID string) string) (oldNodes []corev1.Node, newNodes []corev1.Node) {
	nodeMap := make(map[string]corev1.Node)
	for _, n := range nodes {
		nodeMap[n.GetName()] = n
	}

	for _, i := range instances {
		name := instanceNameFunc(customObject, *i.InstanceID)

		if *i.LatestModelApplied {
			n, found := nodeMap[name]
			// When VMSS is scaling up there might be VM instances that haven't
			// registered as nodes in k8s yet. Hence not all instances are
			// found from node list.
			if found {
				newNodes = append(newNodes, n)
			}
		} else {
			n, found := nodeMap[name]
			if found {
				oldNodes = append(oldNodes, n)
			}
		}
	}

	return
}
