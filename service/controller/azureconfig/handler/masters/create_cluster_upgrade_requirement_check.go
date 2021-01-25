package masters

import (
	"context"

	providerv1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/provider/v1alpha1"
	releasev1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/release/v1alpha1"
	"github.com/giantswarm/conditions/pkg/conditions"
	"github.com/giantswarm/microerror"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v5/pkg/handler/nodes"
	"github.com/giantswarm/azure-operator/v5/pkg/handler/nodes/state"
	"github.com/giantswarm/azure-operator/v5/service/controller/key"
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
		if err != nil {
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

func (r *Resource) getCluster(ctx context.Context, cr *providerv1alpha1.AzureConfig) (*capiv1alpha3.Cluster, error) {
	orgNs := key.OrganizationNamespace(cr)

	cluster := &capiv1alpha3.Cluster{}
	err := r.ctrlClient.Get(ctx, client.ObjectKey{Name: cr.Labels[capiv1alpha3.ClusterLabelName], Namespace: orgNs}, cluster)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return cluster, nil
}
