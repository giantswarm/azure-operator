package instance

import (
	"context"
	"fmt"

	corev1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	"github.com/giantswarm/azure-operator/service/controller/v13/key"
	"github.com/giantswarm/azure-operator/service/controller/v13/resource/instance/internal/state"
	"github.com/giantswarm/microerror"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *Resource) workerInstancesUpgradingTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return "", microerror.Mask(err)
	}

	versionValue := map[string]string{}
	{
		for _, node := range customObject.Status.Cluster.Nodes {
			versionValue[node.Name] = node.Version
		}
	}

	var drainerConfigs []corev1alpha1.DrainerConfig
	{
		n := v1.NamespaceAll
		o := metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s", key.ClusterIDLabel, key.ClusterID(customObject)),
		}

		list, err := r.g8sClient.CoreV1alpha1().DrainerConfigs(n).List(o)
		if err != nil {
			return "", microerror.Mask(err)
		}

		drainerConfigs = list.Items
	}

	var workerUpgradeInProgess bool
	allWorkerInstances, err := r.allInstances(ctx, customObject, key.WorkerVMSSName)
	if IsScaleSetNotFound(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("did not find the scale set '%s'", key.WorkerVMSSName(customObject)))
	} else if err != nil {
		return "", microerror.Mask(err)
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", "processing worker VMSSs")

		ws, err := r.nextInstance(ctx, customObject, allWorkerInstances, drainerConfigs, key.WorkerInstanceName, versionValue)
		if err != nil {
			return "", microerror.Mask(err)
		}
		err = r.updateInstance(ctx, customObject, ws.InstanceToUpdate(), key.WorkerVMSSName, key.WorkerInstanceName)
		if err != nil {
			return "", microerror.Mask(err)
		}
		err = r.createDrainerConfig(ctx, customObject, ws.InstanceToDrain(), key.WorkerInstanceName)
		if err != nil {
			return "", microerror.Mask(err)
		}
		err = r.reimageInstance(ctx, customObject, ws.InstanceToReimage(), key.WorkerVMSSName, key.WorkerInstanceName)
		if err != nil {
			return "", microerror.Mask(err)
		}
		err = r.deleteDrainerConfig(ctx, customObject, ws.InstanceToReimage(), key.WorkerInstanceName, drainerConfigs)
		if err != nil {
			return "", microerror.Mask(err)
		}

		workerUpgradeInProgess = ws.IsWIP()

		r.logger.LogCtx(ctx, "level", "debug", "message", "processed worker VMSSs")
	}

	if !workerUpgradeInProgess {
		return DeploymentCompleted, nil
	}

	// Upgrade still in progress.
	return currentState, nil
}
