package nodes

import (
	"context"

	corev1alpha1 "github.com/giantswarm/apiextensions/v6/pkg/apis/core/v1alpha1"
	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/azure-operator/v6/pkg/label"
)

func (r *Resource) CreateDrainerConfig(ctx context.Context, clusterID, clusterAPIEndpoint string, nodeName string) error {
	r.Logger.Debugf(ctx, "creating drainer config for tenant cluster node")

	c := &corev1alpha1.DrainerConfig{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				label.Cluster: clusterID,
			},
			Name:      nodeName,
			Namespace: clusterID,
		},
		Spec: corev1alpha1.DrainerConfigSpec{
			Guest: corev1alpha1.DrainerConfigSpecGuest{
				Cluster: corev1alpha1.DrainerConfigSpecGuestCluster{
					API: corev1alpha1.DrainerConfigSpecGuestClusterAPI{
						Endpoint: clusterAPIEndpoint,
					},
					ID: clusterID,
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

	err := r.CtrlClient.Create(ctx, c)
	if errors.IsAlreadyExists(err) {
		r.Logger.Debugf(ctx, "did not create drainer config for tenant cluster node")
		r.Logger.Debugf(ctx, "drainer config for tenant cluster node does already exist")
	} else if err != nil {
		return microerror.Mask(err)
	} else {
		r.Logger.Debugf(ctx, "created drainer config for tenant cluster node")
	}

	return nil
}
