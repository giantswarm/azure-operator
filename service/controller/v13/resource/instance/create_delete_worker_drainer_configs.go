package instance

import (
	"context"
	"fmt"

	corev1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	"github.com/giantswarm/microerror"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/azure-operator/service/controller/v13/key"
	"github.com/giantswarm/azure-operator/service/controller/v13/resource/instance/internal/state"
)

func (r *Resource) deleteWorkerDrainerConfigsTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return "", microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "finding all drainerconfigs")

	drainerConfigs := make(map[string]corev1alpha1.DrainerConfig)
	{
		n := metav1.NamespaceAll
		o := metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s", key.ClusterIDLabel, key.ClusterID(customObject)),
		}

		list, err := r.g8sClient.CoreV1alpha1().DrainerConfigs(n).List(o)
		if err != nil {
			return "", microerror.Mask(err)
		}

		for _, dc := range list.Items {
			drainerConfigs[dc.Name] = dc
		}
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found %d drainerconfigs", len(drainerConfigs)))
	r.logger.LogCtx(ctx, "level", "debug", "message", "ensuring that all drainerconfigs have drained condition")

	// Ensure that all DrainerConfigs have DRAINED condition.
	for _, dc := range drainerConfigs {
		if !dc.Status.HasDrainedCondition() && !dc.Status.HasTimeoutCondition() {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("node %s has not been drained yet", dc.Name))
			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")

			// Not all nodes have been drained yet. Resume later.
			return currentState, nil
		}

		if dc.Status.HasTimeoutCondition() {
			r.logger.LogCtx(ctx, "level", "warning", "message", fmt.Sprintf("timeout when draining node %s", dc.Name))
		}
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "ensured that all drainerconfigs have drained condition")
	r.logger.LogCtx(ctx, "level", "debug", "message", "deleting all drainerconfigs")

	// Delete DrainerConfigs now that all nodes have been DRAINED.
	for _, dc := range drainerConfigs {
		err = r.g8sClient.CoreV1alpha1().DrainerConfigs(dc.Namespace).Delete(dc.Name, &metav1.DeleteOptions{})
		if err != nil {
			return "", microerror.Mask(err)
		}
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "deleted all drainerconfigs")

	return TerminateOldWorkerInstances, nil
}
