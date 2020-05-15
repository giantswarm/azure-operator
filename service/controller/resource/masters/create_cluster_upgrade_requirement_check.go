package masters

import (
	"context"

	"github.com/coreos/go-semver/semver"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/azure-operator/v4/pkg/label"
	"github.com/giantswarm/azure-operator/v4/pkg/project"
	"github.com/giantswarm/azure-operator/v4/service/controller/controllercontext"
	"github.com/giantswarm/azure-operator/v4/service/controller/internal/state"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

func (r *Resource) clusterUpgradeRequirementCheckTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return "", microerror.Mask(err)
	}

	isCreating := r.isClusterCreating(cr)
	anyOldNodes, err := r.anyNodesOutOfDate(ctx)
	if IsClientNotFound(err) {
		// The kubernetes API is down.
		// We check if the Legacy Master VMSS exists and in that case
		// we assume this is because we're migrating to Flatcar.
		exists, err := r.vmssExists(ctx, key.ResourceGroupName(cr), key.LegacyMasterVMSSName(cr))
		if err != nil || !exists {
			return "", microerror.Mask(err)
		}
	} else if err != nil {
		return "", microerror.Mask(err)
	}

	if !isCreating && anyOldNodes {
		// Only continue rolling nodes when cluster is not creating and there
		// are old nodes in tenant cluster.
		return MasterInstancesUpgrading, nil
	}

	// Skip instance rolling by default.
	return WaitForRestore, nil
}

func (r *Resource) isClusterCreating(cr providerv1alpha1.AzureConfig) bool {
	// When cluster creation is in the beginning, it doesn't necessarily have
	// any status conditions yet.
	if len(cr.Status.Cluster.Conditions) == 0 {
		return true
	}
	if cr.Status.Cluster.HasCreatingCondition() {
		return true
	}

	return false
}

// anyNodesOutOfDate iterates over all nodes in tenant cluster and finds
// corresponding azure-operator version from node labels. If node doesn't have
// this label or was created with older version than currently reconciling one,
// then this function returns true. Otherwise (including on error) false.
func (r *Resource) anyNodesOutOfDate(ctx context.Context) (bool, error) {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return false, microerror.Mask(err)
	}

	if cc.Client.TenantCluster.K8s == nil {
		return false, clientNotFoundError
	}

	nodeList, err := cc.Client.TenantCluster.K8s.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return false, microerror.Mask(err)
	}

	myVersion := semver.New(project.Version())
	for _, n := range nodeList.Items {
		v, exists := n.GetLabels()[label.OperatorVersion]
		if !exists {
			return true, nil
		}

		nodeVersion := semver.New(v)

		if nodeVersion.LessThan(*myVersion) {
			return true, nil
		}
	}

	return false, nil
}
