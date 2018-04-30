package endpoints

import (
	corev2 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/azure-operator/client"
)

const (
	Name = "endpointsv2"

	httpsPort           = 443
	masterEndpointsName = "master"
)

type Config struct {
	AzureConfig client.AzureConfig
	K8sClient   kubernetes.Interface
	Logger      micrologger.Logger
}

type Resource struct {
	azureConfig client.AzureConfig
	k8sClient   kubernetes.Interface
	logger      micrologger.Logger
}

func New(config Config) (*Resource, error) {
	if err := config.AzureConfig.Validate(); err != nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.AzureConfig.%s", config, err)
	}
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%t.Logger must not be empty", config)
	}

	r := &Resource{
		azureConfig: config.AzureConfig,
		k8sClient:   config.K8sClient,
		logger:      config.Logger,
	}

	return r, nil
}

func (r *Resource) Name() string {
	return Name
}

func toEndpoints(v interface{}) (*corev2.Endpoints, error) {
	if v == nil {
		return nil, nil
	}

	endpoints, ok := v.(*corev2.Endpoints)
	if !ok {
		return nil, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", endpoints, v)
	}

	return endpoints, nil
}
