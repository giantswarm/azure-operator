package clusterupgrade

import (
	"context"
	"fmt"

	"github.com/giantswarm/apiextensions/v3/pkg/label"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	capzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	capiv1alpha3exp "sigs.k8s.io/cluster-api/exp/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v5/pkg/helpers"
	"github.com/giantswarm/azure-operator/v5/pkg/project"
	"github.com/giantswarm/azure-operator/v5/pkg/upgrade"
	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	cr, err := key.ToCluster(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "ensuring that azurecluster has same release label")

	err = r.ensureAzureClusterHasSameRelease(ctx, cr)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "ensured that azurecluster has same release label")

	r.logger.LogCtx(ctx, "level", "debug", "message", "ensuring that machinepools has same release label")

	masterUpgraded, err := r.ensureMasterHasUpgraded(ctx, cr)
	if err != nil {
		return microerror.Mask(err)
	}

	if !masterUpgraded {
		r.logger.LogCtx(ctx, "level", "debug", "message", "master node hasn't upgraded yet")
		r.logger.LogCtx(ctx, "level", "debug", "message", "cancelling resource")
		return nil
	}

	err = r.propagateReleaseToMachinePools(ctx, cr)
	if err != nil {
		return microerror.Mask(err)
	}
	r.logger.LogCtx(ctx, "level", "debug", "message", "ensured that machinepools has same release label")

	return nil
}

func (r *Resource) ensureAzureClusterHasSameRelease(ctx context.Context, cr capiv1alpha3.Cluster) error {
	if cr.Spec.InfrastructureRef == nil {
		return microerror.Maskf(notFoundError, "infrastructure reference not yet set")
	}

	azureCluster := capzv1alpha3.AzureCluster{}
	err := r.ctrlClient.Get(ctx, client.ObjectKey{Namespace: cr.Namespace, Name: cr.Spec.InfrastructureRef.Name}, &azureCluster)
	if err != nil {
		return microerror.Mask(err)
	}

	if cr.Labels[label.ReleaseVersion] == azureCluster.Labels[label.ReleaseVersion] &&
		cr.Labels[label.AzureOperatorVersion] == azureCluster.Labels[label.AzureOperatorVersion] {
		// AzureCluster release & operator version already matches. Nothing to do here.
		return nil
	}

	azureCluster.Labels[label.AzureOperatorVersion] = cr.Labels[label.AzureOperatorVersion]
	azureCluster.Labels[label.ReleaseVersion] = cr.Labels[label.ReleaseVersion]
	err = r.ctrlClient.Update(ctx, &azureCluster)
	if apierrors.IsConflict(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "conflict trying to save object in k8s API concurrently", "stack", microerror.JSON(microerror.Mask(err)))
		r.logger.LogCtx(ctx, "level", "debug", "message", "cancelling resource")
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *Resource) ensureMasterHasUpgraded(ctx context.Context, cluster capiv1alpha3.Cluster) (bool, error) {
	tenantClusterK8sClient, err := r.tenantClientFactory.GetClient(ctx, &cluster)
	if err != nil {
		r.logger.LogCtx(ctx, "level", "debug", "message", "tenant API not available yet", "stack", microerror.JSON(err))
		return false, nil
	}

	nodeList := &corev1.NodeList{}
	err = tenantClusterK8sClient.List(ctx, nodeList, client.MatchingLabels{"kubernetes.io/role": "master"})
	if err != nil {
		return false, microerror.Mask(err)
	}

	for _, n := range nodeList.Items {
		nodeVer, labelExists := n.Labels[label.AzureOperatorVersion]

		if !labelExists || nodeVer != project.Version() {
			return false, nil
		}
	}

	return true, nil
}

func (r *Resource) propagateReleaseToMachinePools(ctx context.Context, cr capiv1alpha3.Cluster) error {
	lst, err := helpers.GetMachinePoolsByMetadata(ctx, r.ctrlClient, cr.ObjectMeta)
	if err != nil {
		return microerror.Mask(err)
	}

	allStartedNodePoolUpgradesAreCompleted := true
	var machinePoolUpgradeCandidate *capiv1alpha3exp.MachinePool

	for _, mp := range lst.Items {
		machinePool := mp

		if cr.Labels[label.ReleaseVersion] == machinePool.Labels[label.ReleaseVersion] &&
			cr.Labels[label.AzureOperatorVersion] == machinePool.Labels[label.AzureOperatorVersion] {
			// Upgrade was initiated for this node pool, let's check if it has
			// been completed.
			nodePoolUpgradeInProgressOrPending, err := upgrade.IsNodePoolUpgradeInProgressOrPending(
				ctx,
				r.ctrlClient,
				&machinePool,
				cr.Labels[label.ReleaseVersion],
				cr.Labels[label.AzureOperatorVersion])
			if err != nil {
				return microerror.Mask(err)
			}

			if nodePoolUpgradeInProgressOrPending {
				// This node pool is still being upgraded, so we want to wait
				// for this upgrade to be completed.
				allStartedNodePoolUpgradesAreCompleted = false
				break
			}
		}

		if machinePoolUpgradeCandidate == nil &&
			(cr.Labels[label.ReleaseVersion] != machinePool.Labels[label.ReleaseVersion] ||
				cr.Labels[label.AzureOperatorVersion] != machinePool.Labels[label.AzureOperatorVersion]) {

			// We did not already find next node pool to be upgraded, and this
			// one does not have version labels updated, so this node pool will
			// be upgraded next.
			machinePoolUpgradeCandidate = &machinePool

			// Only update one MachinePool CR at a time.
			break
		}
	}

	if allStartedNodePoolUpgradesAreCompleted && machinePoolUpgradeCandidate != nil {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updating release to machinepool %q", machinePoolUpgradeCandidate.Name))

		machinePoolUpgradeCandidate.Labels[label.ReleaseVersion] = cr.Labels[label.ReleaseVersion]
		machinePoolUpgradeCandidate.Labels[label.AzureOperatorVersion] = cr.Labels[label.AzureOperatorVersion]

		err = r.ctrlClient.Update(ctx, machinePoolUpgradeCandidate)
		if apierrors.IsConflict(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "conflict trying to save object in k8s API concurrently", "stack", microerror.JSON(microerror.Mask(err)))
			r.logger.LogCtx(ctx, "level", "debug", "message", "cancelling resource")
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updated release to machinepool %q", machinePoolUpgradeCandidate.Name))
	}

	return nil
}