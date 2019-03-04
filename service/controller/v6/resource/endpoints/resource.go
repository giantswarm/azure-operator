package endpoints

import (
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	Name = "endpointsv6"

	httpsPort = 443
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
		return nil, microerror.Maskf(invalidConfigError, "%t.Logger must not be empty", config)
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

func toEndpoints(v interface{}) (*corev1.Endpoints, error) {
	if v == nil {
		return nil, nil
	}

	endpoints, ok := v.(*corev1.Endpoints)
	if !ok {
		return nil, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", endpoints, v)
	}

	return endpoints, nil
}
