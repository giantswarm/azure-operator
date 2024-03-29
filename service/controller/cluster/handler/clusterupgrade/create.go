package clusterupgrade

import (
	"context"

	"github.com/giantswarm/apiextensions/v6/pkg/annotation"
	"github.com/giantswarm/apiextensions/v6/pkg/label"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	capz "sigs.k8s.io/cluster-api-provider-azure/api/v1beta1"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
	capiexp "sigs.k8s.io/cluster-api/exp/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v8/pkg/helpers"
	"github.com/giantswarm/azure-operator/v8/pkg/project"
	"github.com/giantswarm/azure-operator/v8/pkg/tenantcluster"
	"github.com/giantswarm/azure-operator/v8/service/controller/key"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	cr, err := key.ToCluster(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "ensuring that azurecluster has same release label")

	err = r.ensureAzureClusterHasSameRelease(ctx, cr)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "ensured that azurecluster has same release label")

	r.logger.Debugf(ctx, "ensuring that all machinepools has the same release label")

	r.logger.Debugf(ctx, "checking if master has been upgraded already")
	masterUpgraded, err := r.ensureMasterHasUpgraded(ctx, cr)
	if err != nil {
		return microerror.Mask(err)
	}

	if !masterUpgraded {
		r.logger.Debugf(ctx, "master hasn't upgraded yet")
		r.logger.Debugf(ctx, "cancelling resource")
		return nil
	}

	r.logger.Debugf(ctx, "master has been upgraded already")

	err = r.ensureAzureMachineLastReleaseDeployedAnnotation(ctx, &cr)
	if err != nil {
		return microerror.Mask(err)
	}

	machinePoolLst, err := helpers.GetMachinePoolsByMetadata(ctx, r.ctrlClient, cr.ObjectMeta)
	if err != nil {
		return microerror.Mask(err)
	}

	machinePoolUpgrading, err := isAnyMachinePoolUpgrading(cr, machinePoolLst.Items)
	if err != nil {
		return microerror.Mask(err)
	}

	if machinePoolUpgrading {
		r.logger.Debugf(ctx, "there is machinepool upgrading")
		r.logger.Debugf(ctx, "cancelling resource")
		return nil
	}

	r.logger.Debugf(ctx, "finding machinepool that has not been upgraded yet")

	machinePoolsNotUpgradedYet, err := machinePoolsNotUpgradedYet(cr, machinePoolLst.Items)
	if err != nil {
		return microerror.Mask(err)
	}

	if len(machinePoolsNotUpgradedYet) > 0 {
		machinePool := machinePoolsNotUpgradedYet[0]

		r.logger.Debugf(ctx, "found machinepool that has not been upgraded yet: %#q", machinePool.Name)
		r.logger.Debugf(ctx, "updating release & operator version labels")

		machinePool.Labels[label.ReleaseVersion] = cr.Labels[label.ReleaseVersion]
		machinePool.Labels[label.AzureOperatorVersion] = cr.Labels[label.AzureOperatorVersion]
		err = r.ctrlClient.Update(ctx, &machinePool)
		if apierrors.IsConflict(err) {
			r.logger.Debugf(ctx, "conflict trying to save object in k8s API concurrently")
			r.logger.Debugf(ctx, "cancelling resource")
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}

		r.logger.Debugf(ctx, "updated release & operator version labels of machinepool %#q", machinePool.Name)
		r.logger.Debugf(ctx, "cancelling resource")
		return nil
	}

	r.logger.Debugf(ctx, "did not find any machinepool that has not been upgraded yet")
	r.logger.Debugf(ctx, "ensured that all machinepools has the same release label")

	return nil
}

func (r *Resource) ensureAzureClusterHasSameRelease(ctx context.Context, cr capi.Cluster) error {
	if cr.Spec.InfrastructureRef == nil {
		return microerror.Maskf(notFoundError, "infrastructure reference not yet set")
	}

	azureCluster := capz.AzureCluster{}
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
		r.logger.Debugf(ctx, "conflict trying to save object in k8s API concurrently")
		r.logger.Debugf(ctx, "cancelling resource")
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *Resource) ensureMasterHasUpgraded(ctx context.Context, cluster capi.Cluster) (bool, error) {
	tenantClusterK8sClient, err := r.tenantClientFactory.GetClient(ctx, &cluster)
	if tenantcluster.IsAPINotAvailableError(err) {
		r.logger.Debugf(ctx, "tenant API not available yet")
		r.logger.Debugf(ctx, "canceling resource")

		return false, nil
	} else if err != nil {
		return false, microerror.Mask(err)
	}

	nodeList := &corev1.NodeList{}
	err = tenantClusterK8sClient.List(ctx, nodeList, client.MatchingLabels{"kubernetes.io/role": "master"})
	if tenantcluster.IsAPINotAvailableError(err) {
		r.logger.Debugf(ctx, "tenant API not available yet")
		r.logger.Debugf(ctx, "canceling resource")

		return false, nil
	} else if err != nil {
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

func (r *Resource) ensureAzureMachineLastReleaseDeployedAnnotation(ctx context.Context, cluster *capi.Cluster) error {
	r.logger.Debugf(ctx, "ensuring that AzureMachine has annotation %q set to desired release %s", annotation.LastDeployedReleaseVersion, key.ReleaseVersion(cluster))
	azureMachineName := key.AzureMachineName(cluster)
	azureMachine := &capz.AzureMachine{}
	err := r.ctrlClient.Get(ctx, client.ObjectKey{Namespace: cluster.Namespace, Name: azureMachineName}, azureMachine)
	if err != nil {
		return microerror.Mask(err)
	}

	if azureMachine.Annotations == nil {
		azureMachine.Annotations = map[string]string{}
	}

	azureMachine.Annotations[annotation.LastDeployedReleaseVersion] = azureMachine.Labels[label.ReleaseVersion]
	err = r.ctrlClient.Update(ctx, azureMachine)
	if apierrors.IsConflict(err) {
		r.logger.Debugf(ctx, "conflict trying to save object in k8s API concurrently", "stack", microerror.JSON(microerror.Mask(err)))
		r.logger.Debugf(ctx, "cancelling resource")
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "ensured that AzureMachine has annotation %s set to desired release %s", annotation.LastDeployedReleaseVersion, key.ReleaseVersion(cluster))
	return nil
}

func machinePoolsNotUpgradedYet(cr capi.Cluster, machinePools []capiexp.MachinePool) ([]capiexp.MachinePool, error) {
	desiredRelease := cr.Labels[label.ReleaseVersion]

	var pendingUpgrade []capiexp.MachinePool

	for i := range machinePools {
		machinePool := machinePools[i]

		isUpgrading, err := isMachinePoolUpgradingInProgress(&machinePool, desiredRelease)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		hasUpgraded, err := isMachinePoolUpgraded(&machinePool, desiredRelease)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		if !isUpgrading && !hasUpgraded {
			pendingUpgrade = append(pendingUpgrade, machinePool)
		}
	}

	return pendingUpgrade, nil
}
