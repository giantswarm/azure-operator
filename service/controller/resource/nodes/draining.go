package nodes

import (
	"context"

	corev1alpha1 "github.com/giantswarm/apiextensions/v2/pkg/apis/core/v1alpha1"
	providerv1alpha1 "github.com/giantswarm/apiextensions/v2/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/azure-operator/v4/pkg/label"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

func (r *Resource) CreateDrainerConfig(ctx context.Context, customObject providerv1alpha1.AzureConfig, nodeName string) error {
	r.Logger.LogCtx(ctx, "level", "debug", "message", "creating drainer config for tenant cluster node")

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

	_, err := r.G8sClient.CoreV1alpha1().DrainerConfigs(n).Create(ctx, c, metav1.CreateOptions{})
	if errors.IsAlreadyExists(err) {
		r.Logger.LogCtx(ctx, "level", "debug", "message", "did not create drainer config for tenant cluster node")
		r.Logger.LogCtx(ctx, "level", "debug", "message", "drainer config for tenant cluster node does already exist")
	} else if err != nil {
		return microerror.Mask(err)
	} else {
		r.Logger.LogCtx(ctx, "level", "debug", "message", "created drainer config for tenant cluster node")
	}

	return nil
}
