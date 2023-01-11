package namespace

import (
	"context"

	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	apismetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/azure-operator/v7/service/controller/key"
)

func (r *Resource) GetDesiredState(ctx context.Context, obj interface{}) (interface{}, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	namespace := &corev1.Namespace{
		TypeMeta: apismetav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		ObjectMeta: apismetav1.ObjectMeta{
			Name: key.ClusterNamespace(cr),
			Labels: map[string]string{
				key.LabelApp:           "master",
				key.LegacyLabelCluster: key.ClusterID(&cr),
				key.LabelCustomer:      key.ClusterCustomer(cr),
				key.LabelCluster:       key.ClusterID(&cr),
				key.LabelOrganization:  key.ClusterCustomer(cr),
			},
		},
	}

	return namespace, nil
}
