package service

import (
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"reflect"
)

const (
	Name = "servicev3"

	httpsPort         = 443
	masterServiceName = "master"
)

type Config struct {
	K8sClient kubernetes.Interface
	Logger    micrologger.Logger
}

type Resource struct {
	k8sClient kubernetes.Interface
	logger    micrologger.Logger
}

func New(config Config) (*Resource, error) {
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	r := &Resource{
		k8sClient: config.K8sClient,
		logger:    config.Logger,
	}

	return r, nil
}

func (r *Resource) Name() string {
	return Name
}

func getServiceByName(list []*corev1.Service, name string) (*corev1.Service, error) {
	for _, l := range list {
		if l.Name == name {
			return l, nil
		}
	}

	return nil, microerror.Mask(notFoundError)
}

func isServiceModified(a, b *corev1.Service) bool {
	if !reflect.DeepEqual(a.Spec, b.Spec) {
		return true
	}

	if !reflect.DeepEqual(a.Labels, b.Labels) {
		return true
	}

	if !reflect.DeepEqual(a.Annotations, b.Annotations) {
		return true
	}

	return false
}

func toService(v interface{}) (*corev1.Service, error) {
	if v == nil {
		return nil, nil
	}

	service, ok := v.(*corev1.Service)
	if !ok {
		return nil, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", &corev1.Service{}, v)
	}

	return service, nil
}
func toServices(v interface{}) ([]*corev1.Service, error) {
	if v == nil {
		return nil, nil
	}

	services, ok := v.([]*corev1.Service)
	if !ok {
		return nil, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", []*corev1.Service{}, v)
	}

	return services, nil
}
