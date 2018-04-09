package service

import (
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	Name = "servicev1"

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

func toService(v interface{}) (*apiv1.Service, error) {
	if v == nil {
		return nil, nil
	}

	service, ok := v.(*apiv1.Service)
	if !ok {
		return nil, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", &apiv1.Service{}, v)
	}

	return service, nil
}
