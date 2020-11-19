package clusterupgrade

import (
	"context"
	"fmt"

	"github.com/giantswarm/apiextensions/v3/pkg/label"
	"github.com/giantswarm/errors/tenant"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	capzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	expcapiv1alpha3 "sigs.k8s.io/cluster-api/exp/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v5/pkg/conditions"
	"github.com/giantswarm/azure-operator/v5/pkg/helpers"
	"github.com/giantswarm/azure-operator/v5/pkg/project"
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

	r.logger.LogCtx(ctx, "level", "debug", "message", "ensuring that all machinepools has the same release label")

	r.logger.LogCtx(ctx, "level", "debug", "message", "checking if master has been upgraded already")
	masterUpgraded, err := r.ensureMasterHasUpgraded(ctx, cr)
	if err != nil {
		return microerror.Mask(err)
	}

	if !masterUpgraded {
		r.logger.LogCtx(ctx, "level", "debug", "message", "master hasn't upgraded yet")
		r.logger.LogCtx(ctx, "level", "debug", "message", "cancelling resource")
		return nil
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "master has been upgraded already")

	machinePoolLst, err := helpers.GetMachinePoolsByMetadata(ctx, r.ctrlClient, cr.ObjectMeta)
	if err != nil {
		return microerror.Mask(err)
	}

	machinePoolUpgrading, err := isAnyMachinePoolUpgrading(cr, machinePoolLst.Items)
	if err != nil {
		return microerror.Mask(err)
	}

	if machinePoolUpgrading {
		r.logger.LogCtx(ctx, "level", "debug", "message", "there is machinepool upgrading")
		r.logger.LogCtx(ctx, "level", "debug", "message", "cancelling resource")
		return nil
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "finding machinepool that has not been upgraded yet")

	machinePoolsNotUpgradedYet, err := machinePoolsNotUpgradedYet(cr, machinePoolLst.Items)
	if err != nil {
		return microerror.Mask(err)
	}

	if len(machinePoolsNotUpgradedYet) > 0 {
		machinePool := machinePoolsNotUpgradedYet[0]

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found machinepool that has not been upgraded yet: %#q", machinePool.Name))
		r.logger.LogCtx(ctx, "level", "debug", "message", "updating release & operator version labels")

		machinePool.Labels[label.ReleaseVersion] = cr.Labels[label.ReleaseVersion]
		machinePool.Labels[label.AzureOperatorVersion] = cr.Labels[label.AzureOperatorVersion]
		err = r.ctrlClient.Update(ctx, &machinePool)
		if apierrors.IsConflict(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "conflict trying to save object in k8s API concurrently", "stack", microerror.JSON(microerror.Mask(err)))
			r.logger.LogCtx(ctx, "level", "debug", "message", "cancelling resource")
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updated release & operator version labels of machinepool %#q", machinePool.Name))
		r.logger.LogCtx(ctx, "level", "debug", "message", "cancelling resource")
		return nil
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "did not find any machinepool that has not been upgraded yet")
	r.logger.LogCtx(ctx, "level", "debug", "message", "ensured that all machinepools has the same release label")

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
	if tenant.IsAPINotAvailable(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "tenant API not available yet")
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")

		return false, nil
	} else if err != nil {
		return false, microerror.Mask(err)
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

func isAnyMachinePoolUpgrading(cr capiv1alpha3.Cluster, machinePools []expcapiv1alpha3.MachinePool) (bool, error) {
	desiredRelease := cr.Labels[label.ReleaseVersion]

	for _, machinePool := range machinePools {
		isUpgrading, err := conditions.IsUpgradingInProgress(&machinePool, desiredRelease)
		if err != nil {
			return false, microerror.Mask(err)
		}

		if isUpgrading {
			return true, nil
		}
	}

	return false, nil
}

func machinePoolsNotUpgradedYet(cr capiv1alpha3.Cluster, machinePools []expcapiv1alpha3.MachinePool) ([]expcapiv1alpha3.MachinePool, error) {
	desiredRelease := cr.Labels[label.ReleaseVersion]

	var pendingUpgrade []expcapiv1alpha3.MachinePool

	for _, machinePool := range machinePools {
		isUpgrading, err := conditions.IsUpgradingInProgress(&machinePool, desiredRelease)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		hasUpgraded, err := conditions.IsUpgraded(&machinePool, desiredRelease)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		if !isUpgrading && !hasUpgraded {
			pendingUpgrade = append(pendingUpgrade, machinePool)
		}
	}

	return pendingUpgrade, nil
}
