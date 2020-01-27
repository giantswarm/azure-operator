package instance

import (
	"context"
	"fmt"

	corev1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	"github.com/giantswarm/azure-operator/service/controller/v13/key"
	"github.com/giantswarm/azure-operator/service/controller/v13/resource/instance/internal/state"
	"github.com/giantswarm/microerror"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *Resource) createWorkerDrainerConfigsTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return "", microerror.Mask(err)
	}

	var drainerConfigs map[string]corev1alpha1.DrainerConfig
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

	allWorkerInstances, err := r.allInstances(ctx, customObject, key.WorkerVMSSName)
	if IsScaleSetNotFound(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("did not find the scale set '%s'", key.WorkerVMSSName(customObject)))
	} else if err != nil {
		return "", microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "processing worker VMSSs")

	for _, i := range allWorkerInstances {
		if *i.LatestModelApplied {
			continue
		}

		n, err := key.WorkerInstanceName(customObject, *i.InstanceID)
		if err != nil {
			return "", microerror.Mask(err)
		}

		_, drainerConfigExists := drainerConfigs[n]
		if drainerConfigExists {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("DrainerConfig for %s already exists", n))
			continue
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating DrainerConfig for %s", n))
		err = r.createDrainerConfig(ctx, customObject, &i, key.WorkerInstanceName)
		if err != nil {
			return "", microerror.Mask(err)
		}
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "processed worker VMSSs")

	return DeleteWorkerDrainerConfigs, nil
}
