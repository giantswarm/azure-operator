package namespace

import (
	"context"

	corev2 "k8s.io/api/core/v1"
	apismetav2 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/service/controller/v2/key"
)

func (r *Resource) GetDesiredState(ctx context.Context, obj interface{}) (interface{}, error) {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	namespace := &corev2.Namespace{
		TypeMeta: apismetav2.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v2",
		},
		ObjectMeta: apismetav2.ObjectMeta{
			Name: key.ClusterNamespace(customObject),
			Labels: map[string]string{
				"cluster":  key.ClusterID(customObject),
				"customer": key.ClusterCustomer(customObject),
			},
		},
	}

	return namespace, nil
}
