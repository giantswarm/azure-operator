package masters

import (
	"context"

	"github.com/giantswarm/errors/tenant"
	"github.com/giantswarm/tenantcluster/v6/pkg/tenantcluster"

	providerv1alpha1 "github.com/giantswarm/apiextensions/v6/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/conditions/pkg/conditions"
	"github.com/giantswarm/microerror"
	releasev1alpha1 "github.com/giantswarm/release-operator/v4/api/v1alpha1"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v7/pkg/handler/nodes"
	"github.com/giantswarm/azure-operator/v7/pkg/handler/nodes/state"
	"github.com/giantswarm/azure-operator/v7/service/controller/key"
)

func (r *Resource) clusterUpgradeRequirementCheckTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return "", microerror.Mask(err)
	}

	var releases []releasev1alpha1.Release
	{
		var rels releasev1alpha1.ReleaseList
		err := r.ctrlClient.List(ctx, &rels)
		if err != nil {
			return "", microerror.Mask(err)
		}
		releases = rels.Items
	}

	var tenantClusterK8sClient client.Client
	{
		tenantClusterK8sClient, err = r.getTenantClusterClient(ctx, &cr)
		if tenant.IsAPINotAvailable(err) || tenantcluster.IsTimeout(err) {
			// The kubernetes API is not reachable. This usually happens when a new cluster is being created.
			// This makes the whole controller to fail and stops next handlers from being executed even if they are
			// safe to run. We don't want that to happen so we just return and we'll try again during next loop.
			r.Logger.Debugf(ctx, "tenant API not available yet")
			r.Logger.Debugf(ctx, "canceling resource")

			return currentState, nil
		} else if err != nil {
			return "", microerror.Mask(err)
		}
	}

	anyOldNodes, err := nodes.AnyOutOfDate(ctx, tenantClusterK8sClient, key.ReleaseVersion(&cr), releases, map[string]string{"role": "master"})
	if nodes.IsClientNotFound(err) {
		r.Logger.Debugf(ctx, "tenant cluster client not found")
		return currentState, nil
	} else if err != nil {
		return "", microerror.Mask(err)
	}

	cluster, err := r.getCluster(ctx, &cr)
	if err != nil {
		return "", microerror.Mask(err)
	}

	if conditions.IsCreatingFalse(cluster) && anyOldNodes {
		// Only continue rolling nodes when cluster is not creating and there
		// are old nodes in tenant cluster.
		return MasterInstancesUpgrading, nil
	}

	// Skip instance rolling by default.
	return DeploymentCompleted, nil
}

func (r *Resource) getCluster(ctx context.Context, cr *providerv1alpha1.AzureConfig) (*capi.Cluster, error) {
	orgNs := key.OrganizationNamespace(cr)

	cluster := &capi.Cluster{}
	err := r.ctrlClient.Get(ctx, client.ObjectKey{Name: cr.Labels[capi.ClusterLabelName], Namespace: orgNs}, cluster)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return cluster, nil
}
