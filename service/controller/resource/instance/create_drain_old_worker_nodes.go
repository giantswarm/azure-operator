package instance

import (
	"context"
	"fmt"

	corev1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/azure-operator/v4/pkg/label"
	"github.com/giantswarm/azure-operator/v4/service/controller/internal/state"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

func (r *Resource) drainOldWorkerNodesTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return "", microerror.Mask(err)
	}

	r.Logger().LogCtx(ctx, "level", "debug", "message", "finding all drainerconfigs")

	drainerConfigs := make(map[string]corev1alpha1.DrainerConfig)
	{
		n := metav1.NamespaceAll
		o := metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s", label.Cluster, key.ClusterID(&cr)),
		}

		list, err := r.G8sClient().CoreV1alpha1().DrainerConfigs(n).List(o)
		if err != nil {
			return "", microerror.Mask(err)
		}

		for _, dc := range list.Items {
			drainerConfigs[dc.Name] = dc
		}
	}

	r.Logger().LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found %d drainerconfigs", len(drainerConfigs)))
	r.Logger().LogCtx(ctx, "level", "debug", "message", "finding all worker VMSS instances")

	allWorkerInstances, err := r.allInstances(ctx, cr, key.WorkerVMSSName)
	if IsScaleSetNotFound(err) {
		r.Logger().LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("did not find the scale set '%s'", key.WorkerVMSSName(cr)))
	} else if err != nil {
		return "", microerror.Mask(err)
	}

	r.Logger().LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found %d worker VMSS instances", len(allWorkerInstances)))
	r.Logger().LogCtx(ctx, "level", "debug", "message", "ensuring that drainerconfig exists for all old worker nodes")

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
			r.Logger().LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating drainerconfig for %s", n))
			err = r.createDrainerConfig(ctx, cr, key.WorkerInstanceName(cr, *i.InstanceID))
			if err != nil {
				return "", microerror.Mask(err)
			}
		}

		if drainerConfigExists && dc.Status.HasTimeoutCondition() {
			nodesPendingDraining++
			r.Logger().LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("drainerconfig for %s already exists but has timed out", n))
			r.Logger().LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deleting drainerconfig for %s", n))

			err = r.G8sClient().CoreV1alpha1().DrainerConfigs(dc.Namespace).Delete(dc.Name, &metav1.DeleteOptions{})
			if errors.IsNotFound(err) {
				r.Logger().LogCtx(ctx, "level", "debug", "message", "did not delete drainer config for tenant cluster node")
				r.Logger().LogCtx(ctx, "level", "debug", "message", "drainer config for tenant cluster node does not exist")
			} else if err != nil {
				return "", microerror.Mask(err)
			}

			r.Logger().LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating drainerconfig for %s", n))
			err = r.createDrainerConfig(ctx, cr, key.WorkerInstanceName(cr, *i.InstanceID))
			if err != nil {
				return "", microerror.Mask(err)
			}
		}

		if drainerConfigExists && !dc.Status.HasTimeoutCondition() && !dc.Status.HasDrainedCondition() {
			nodesPendingDraining++
			r.Logger().LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("drainerconfig for %s already exists", n))
		}
	}

	r.Logger().LogCtx(ctx, "level", "debug", "message", "ensured that drainerconfig exists for all old worker nodes")
	r.Logger().LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("%d nodes are pending draining", nodesPendingDraining))

	if nodesPendingDraining > 0 {
		r.Logger().LogCtx(ctx, "level", "debug", "message", "cancelling resource")
		return currentState, nil
	}

	r.Logger().LogCtx(ctx, "level", "debug", "message", "deleting all drainerconfigs")

	// Delete DrainerConfigs now that all nodes have been DRAINED.
	for _, dc := range drainerConfigs {
		err = r.G8sClient().CoreV1alpha1().DrainerConfigs(dc.Namespace).Delete(dc.Name, &metav1.DeleteOptions{})
		if err != nil {
			return "", microerror.Mask(err)
		}
	}

	r.Logger().LogCtx(ctx, "level", "debug", "message", "deleted all drainerconfigs")

	return TerminateOldWorkerInstances, nil
}

func (r *Resource) createDrainerConfig(ctx context.Context, customObject providerv1alpha1.AzureConfig, nodeName string) error {
	r.Logger().LogCtx(ctx, "level", "debug", "message", "creating drainer config for tenant cluster node")

	n := key.ClusterID(&customObject)
	c := &corev1alpha1.DrainerConfig{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				label.Cluster: key.ClusterID(&customObject),
			},
			Name: nodeName,
		},
		Spec: corev1alpha1.DrainerConfigSpec{
			Guest: corev1alpha1.DrainerConfigSpecGuest{
				Cluster: corev1alpha1.DrainerConfigSpecGuestCluster{
					API: corev1alpha1.DrainerConfigSpecGuestClusterAPI{
						Endpoint: key.ClusterAPIEndpoint(customObject),
					},
					ID: key.ClusterID(&customObject),
				},
				Node: corev1alpha1.DrainerConfigSpecGuestNode{
					Name: nodeName,
				},
			},
			VersionBundle: corev1alpha1.DrainerConfigSpecVersionBundle{
				Version: "0.2.0",
			},
		},
	}

	_, err := r.G8sClient().CoreV1alpha1().DrainerConfigs(n).Create(c)
	if errors.IsAlreadyExists(err) {
		r.Logger().LogCtx(ctx, "level", "debug", "message", "did not create drainer config for tenant cluster node")
		r.Logger().LogCtx(ctx, "level", "debug", "message", "drainer config for tenant cluster node does already exist")
	} else if err != nil {
		return microerror.Mask(err)
	} else {
		r.Logger().LogCtx(ctx, "level", "debug", "message", "created drainer config for tenant cluster node")
	}

	return nil
}
