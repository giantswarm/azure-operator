package instance

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	corev1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/azure-operator/service/controller/v12/key"
	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *Resource) createDrainerConfig(ctx context.Context, customObject providerv1alpha1.AzureConfig, instance *compute.VirtualMachineScaleSetVM, instanceNameFunc func(customObject providerv1alpha1.AzureConfig, instanceID string) (string, error)) error {
	if instance == nil {
		return nil
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "creating drainer config for tenant cluster node")

	instanceName, err := instanceNameFunc(customObject, *instance.InstanceID)
	if err != nil {
		return microerror.Mask(err)
	}

	n := key.ClusterID(customObject)
	c := &corev1alpha1.DrainerConfig{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				key.ClusterIDLabel: key.ClusterID(customObject),
			},
			Name: instanceName,
		},
		Spec: corev1alpha1.DrainerConfigSpec{
			Guest: corev1alpha1.DrainerConfigSpecGuest{
				Cluster: corev1alpha1.DrainerConfigSpecGuestCluster{
					API: corev1alpha1.DrainerConfigSpecGuestClusterAPI{
						Endpoint: key.ClusterAPIEndpoint(customObject),
					},
					ID: key.ClusterID(customObject),
				},
				Node: corev1alpha1.DrainerConfigSpecGuestNode{
					Name: instanceName,
				},
			},
			VersionBundle: corev1alpha1.DrainerConfigSpecVersionBundle{
				Version: "0.2.0",
			},
		},
	}

	_, err = r.g8sClient.CoreV1alpha1().DrainerConfigs(n).Create(c)
	if errors.IsAlreadyExists(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "did not create drainer config for tenant cluster node")
		r.logger.LogCtx(ctx, "level", "debug", "message", "drainer config for tenant cluster node does already exist")
	} else if err != nil {
		return microerror.Mask(err)
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", "created drainer config for tenant cluster node")
	}

	return nil
}

func (r *Resource) deleteDrainerConfig(ctx context.Context, customObject providerv1alpha1.AzureConfig, instance *compute.VirtualMachineScaleSetVM, instanceNameFunc func(customObject providerv1alpha1.AzureConfig, instanceID string) (string, error), drainerConfigs []corev1alpha1.DrainerConfig) error {
	if instance == nil {
		return nil
	}

	instanceName, err := instanceNameFunc(customObject, *instance.InstanceID)
	if err != nil {
		return microerror.Mask(err)
	}

	if isNodeDrained(drainerConfigs, instanceName) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "deleting drainer config for tenant cluster node")

		var drainerConfigToRemove corev1alpha1.DrainerConfig
		for _, n := range drainerConfigs {
			if n.GetName() == instanceName {
				drainerConfigToRemove = n
				break
			}
		}

		n := drainerConfigToRemove.GetNamespace()
		i := drainerConfigToRemove.GetName()
		o := &metav1.DeleteOptions{}

		err := r.g8sClient.CoreV1alpha1().DrainerConfigs(n).Delete(i, o)
		if errors.IsNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "did not delete drainer config for tenant cluster node")
			r.logger.LogCtx(ctx, "level", "debug", "message", "drainer config for tenant cluster node does not exist")
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", "deleted drainer config for tenant cluster node")
		}
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", "not deleting drainer config for tenant cluster node due to undrained node")
	}

	// TODO implement safety net to delete drainer configs that are over due for e.g. when node-operator fucks up

	return nil
}

func isNodeDrained(drainerConfigs []corev1alpha1.DrainerConfig, instanceName string) bool {
	for _, n := range drainerConfigs {
		if n.GetName() != instanceName {
			continue
		}
		if n.Status.HasDrainedCondition() {
			return true
		}
		if n.Status.HasTimeoutCondition() {
			return true
		}
	}

	return false
}
