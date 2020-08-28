package nodepool

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/coreos/go-semver/semver"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/cluster-api/util"

	"github.com/giantswarm/azure-operator/v4/pkg/label"
	"github.com/giantswarm/azure-operator/v4/pkg/project"
	"github.com/giantswarm/azure-operator/v4/service/controller/controllercontext"
	"github.com/giantswarm/azure-operator/v4/service/controller/internal/state"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

const (
	// UnschedulablePatch is the JSON patch structure being applied to nodes using
	// a strategic merge patch in order to cordon them.
	UnschedulablePatch = `{"spec":{"unschedulable":true}}`
)

func (r *Resource) cordonOldWorkersTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	azureMachinePool, err := key.ToAzureMachinePool(obj)
	if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}

	cluster, err := util.GetClusterFromMetadata(ctx, r.CtrlClient, azureMachinePool.ObjectMeta)
	if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}

	credentialSecret, err := r.getCredentialSecret(ctx, *cluster)
	if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}

	virtualMachineScaleSetVMsClient, err := r.ClientFactory.GetVirtualMachineScaleSetVMsClient(credentialSecret.Namespace, credentialSecret.Name)
	if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}

	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}

	if cc.Client.TenantCluster.K8s == nil {
		r.Logger.LogCtx(ctx, "level", "debug", "message", "tenant cluster client not available yet")
		return currentState, nil
	}

	var allWorkerInstances []compute.VirtualMachineScaleSetVM
	{
		r.Logger.LogCtx(ctx, "level", "debug", "message", "finding all worker VMSS instances")

		allWorkerInstances, err = r.AllWorkerInstances(ctx, virtualMachineScaleSetVMsClient, key.ClusterID(&azureMachinePool), key.NodePoolVMSSName(&azureMachinePool))
		if err != nil {
			return DeploymentUninitialized, microerror.Mask(err)
		}

		r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found %d worker VMSS instances", len(allWorkerInstances)))
	}

	r.Logger.LogCtx(ctx, "level", "debug", "message", "finding all tenant cluster nodes")

	var nodes []corev1.Node
	{
		nodeList, err := cc.Client.TenantCluster.K8s.CoreV1().Nodes().List(metav1.ListOptions{})
		if err != nil {
			return DeploymentUninitialized, microerror.Mask(err)
		}
		nodes = nodeList.Items
	}

	oldNodes, newNodes := sortNodesByTenantVMState(nodes, allWorkerInstances, key.ClusterID(&azureMachinePool), key.WorkerInstanceName)
	if len(newNodes) < len(oldNodes) {
		// Wait until there's enough new nodes up.
		r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("number of new nodes (%d) is smaller than number of old nodes (%d)", len(newNodes), len(oldNodes)))
		r.Logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return currentState, nil
	}

	r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found %d old and %d new nodes from tenant cluster", len(oldNodes), len(newNodes)))
	r.Logger.LogCtx(ctx, "level", "debug", "message", "ensuring old nodes are cordoned")

	oldNodesCordoned, err := r.ensureNodesCordoned(ctx, oldNodes)
	if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}

	if oldNodesCordoned < len(oldNodes) {
		r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("not all old nodes are still cordoned; %d pending", len(oldNodes)-oldNodesCordoned))

		return currentState, nil
	}

	r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensured all old nodes (%d) are cordoned", oldNodesCordoned))

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
			// On manual operations or during auto-scaling it may happen that
			// node gets terminated while instances are processed. It's ok from
			// cordoning point of view since the node would get deleted later
			// anyway.
		} else if err != nil {
			return 0, microerror.Mask(err)
		}

		count++
	}

	return count, nil
}

func sortNodesByTenantVMState(nodes []corev1.Node, instances []compute.VirtualMachineScaleSetVM, clusterID string, instanceNameFunc func(clusterID, instanceID string) string) (oldNodes []corev1.Node, newNodes []corev1.Node) {
	nodeMap := make(map[string]corev1.Node)
	for _, n := range nodes {
		nodeMap[n.GetName()] = n
	}

	myVersion := semver.New(project.Version())

	for _, i := range instances {
		name := instanceNameFunc(clusterID, *i.InstanceID)

		n, found := nodeMap[name]
		if !found {
			// When VMSS is scaling up there might be VM instances that haven't
			// registered as nodes in k8s yet. Hence not all instances are
			// found from node list.
			continue
		}

		v, exists := n.GetLabels()[label.OperatorVersion]
		if !exists {
			// Label does not exist, this normally happens when a new node is coming up but did not finish
			// its kubernetes bootstrap yet and thus doesn't have all the needed labels.
			// We'll ignore this node for now and wait for it to bootstrap correctly.
			continue
		}

		nodeVersion := semver.New(v)
		if nodeVersion.LessThan(*myVersion) {
			oldNodes = append(oldNodes, n)
		} else {
			newNodes = append(newNodes, n)
		}
	}

	return
}

func (r *Resource) getK8sWorkerNodeForInstance(ctx context.Context, clusterID string, instance compute.VirtualMachineScaleSetVM) (*corev1.Node, error) {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	name := key.WorkerInstanceName(clusterID, *instance.InstanceID)

	nodeList, err := cc.Client.TenantCluster.K8s.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return nil, microerror.Mask(err)
	}
	nodes := nodeList.Items

	for _, n := range nodes {
		if n.GetName() == name {
			return &n, nil
		}
	}

	// Node related to this instance was not found.
	return nil, nil
}

func (r *Resource) isWorkerInstanceFromPreviousRelease(ctx context.Context, clusterID string, instance compute.VirtualMachineScaleSetVM) (*bool, error) {
	t := true
	f := false

	n, err := r.getK8sWorkerNodeForInstance(ctx, clusterID, instance)
	if err != nil {
		return nil, err
	}

	if n == nil {
		// Kubernetes node related to this instance not found, we consider the node old.
		return &t, nil
	}

	myVersion := semver.New(project.Version())

	v, exists := n.GetLabels()[label.OperatorVersion]
	if !exists {
		// Label does not exist, this normally happens when a new node is coming up but did not finish
		// its kubernetes bootstrap yet and thus doesn't have all the needed labels.
		// We'll ignore this node for now and wait for it to bootstrap correctly.
		return nil, nil
	}

	nodeVersion := semver.New(v)
	if nodeVersion.LessThan(*myVersion) {
		return &t, nil
	} else {
		return &f, nil
	}
}