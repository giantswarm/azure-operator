package service

import (
	"context"

	"github.com/giantswarm/microerror"
	v1 "k8s.io/api/core/v1"
	apismetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/giantswarm/azure-operator/v8/service/controller/key"
)

func (r *Resource) GetDesiredState(ctx context.Context, obj interface{}) (interface{}, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	service := &v1.Service{
		ObjectMeta: apismetav1.ObjectMeta{
			Name:      "master",
			Namespace: key.ClusterID(&cr),
			Labels: map[string]string{
				key.LabelApp:           "master",
				key.LegacyLabelCluster: key.ClusterID(&cr),
				key.LabelCustomer:      key.ClusterCustomer(cr),
				key.LabelCluster:       key.ClusterID(&cr),
				key.LabelOrganization:  key.ClusterCustomer(cr),
			},
			Annotations: map[string]string{
				key.AnnotationPrometheusCluster: key.ClusterID(&cr),
				key.AnnotationEtcdDomain:        key.ClusterEtcdDomain(cr),
			},
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Protocol:   v1.ProtocolTCP,
					Port:       httpsPort,
					TargetPort: intstr.FromInt(httpsPort),
				},
			},
		},
	}

	return service, nil
}
