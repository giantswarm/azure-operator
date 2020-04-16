package instance

import (
	"context"
	"fmt"

	corev1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/azure-operator/service/controller/key"
	"github.com/giantswarm/azure-operator/service/controller/resource/instance/internal/state"
)

func (r *Resource) drainOldWorkerNodesTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return "", microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "finding all drainerconfigs") // nolint: errcheck

	drainerConfigs := make(map[string]corev1alpha1.DrainerConfig)
	{
		n := metav1.NamespaceAll
		o := metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s", key.ClusterIDLabel, key.ClusterID(cr)),
		}

		list, err := r.g8sClient.CoreV1alpha1().DrainerConfigs(n).List(o)
		if err != nil {
			return "", microerror.Mask(err)
		}

		for _, dc := range list.Items {
			drainerConfigs[dc.Name] = dc
		}
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found %d drainerconfigs", len(drainerConfigs))) // nolint: errcheck
	r.logger.LogCtx(ctx, "level", "debug", "message", "finding all worker VMSS instances")                         // nolint: errcheck

	allWorkerInstances, err := r.allInstances(ctx, cr, key.WorkerVMSSName)
	if IsScaleSetNotFound(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("did not find the scale set '%s'", key.WorkerVMSSName(cr))) // nolint: errcheck
	} else if err != nil {
		return "", microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found %d worker VMSS instances", len(allWorkerInstances))) // nolint: errcheck
	r.logger.LogCtx(ctx, "level", "debug", "message", "ensuring that drainerconfig exists for all old worker nodes")          // nolint: errcheck

	var nodesPendingDraining int
	for _, i := range allWorkerInstances {
		old, err := r.isWorkerInstanceFromPreviousRelease(ctx, cr, i)
		if err != nil {
			return DeploymentUninitialized, nil
		}

		if old == nil || !*old {
			// Node is a new one or we weren't able to check it's status, don't drain it.
			continue
		}

		n := key.WorkerInstanceName(cr, *i.InstanceID)

		dc, drainerConfigExists := drainerConfigs[n]
		if !drainerConfigExists {
			nodesPendingDraining++
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating drainerconfig for %s", n)) // nolint: errcheck
			err = r.createDrainerConfig(ctx, cr, key.WorkerInstanceName(cr, *i.InstanceID))
			if err != nil {
				return "", microerror.Mask(err)
			}
		}

		if drainerConfigExists && dc.Status.HasTimeoutCondition() {
			nodesPendingDraining++
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("drainerconfig for %s already exists but has timed out", n)) // nolint: errcheck
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deleting drainerconfig for %s", n))                         // nolint: errcheck

			err = r.g8sClient.CoreV1alpha1().DrainerConfigs(dc.Namespace).Delete(dc.Name, &metav1.DeleteOptions{})
			if errors.IsNotFound(err) {
				r.logger.LogCtx(ctx, "level", "debug", "message", "did not delete drainer config for tenant cluster node") // nolint: errcheck
				r.logger.LogCtx(ctx, "level", "debug", "message", "drainer config for tenant cluster node does not exist") // nolint: errcheck
			} else if err != nil {
				return "", microerror.Mask(err)
			}

			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating drainerconfig for %s", n)) // nolint: errcheck
			err = r.createDrainerConfig(ctx, cr, key.WorkerInstanceName(cr, *i.InstanceID))
			if err != nil {
				return "", microerror.Mask(err)
			}
		}

		if drainerConfigExists && !dc.Status.HasTimeoutCondition() && !dc.Status.HasDrainedCondition() {
			nodesPendingDraining++
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("drainerconfig for %s already exists", n)) // nolint: errcheck
		}
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "ensured that drainerconfig exists for all old worker nodes")       // nolint: errcheck
	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("%d nodes are pending draining", nodesPendingDraining)) // nolint: errcheck

	if nodesPendingDraining > 0 {
		r.logger.LogCtx(ctx, "level", "debug", "message", "cancelling resource") // nolint: errcheck
		return currentState, nil
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "deleting all drainerconfigs") // nolint: errcheck

	// Delete DrainerConfigs now that all nodes have been DRAINED.
	for _, dc := range drainerConfigs {
		err = r.g8sClient.CoreV1alpha1().DrainerConfigs(dc.Namespace).Delete(dc.Name, &metav1.DeleteOptions{})
		if err != nil {
			return "", microerror.Mask(err)
		}
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "deleted all drainerconfigs") // nolint: errcheck

	return TerminateOldWorkerInstances, nil
}
