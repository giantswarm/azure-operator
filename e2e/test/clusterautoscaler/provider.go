// +build k8srequired

package clusterautoscaler

import (
	"github.com/giantswarm/apiextensions/v2/pkg/clientset/versioned"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
)

type ProviderConfig struct {
	G8sClient versioned.Interface
	Logger    micrologger.Logger
}

type Provider struct {
	g8sClient versioned.Interface
	logger    micrologger.Logger
}

func NewProvider(config ProviderConfig) (*Provider, error) {
	if config.G8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.G8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	p := &Provider{
		g8sClient: config.G8sClient,
		logger:    config.Logger,
	}

	return p, nil
}
