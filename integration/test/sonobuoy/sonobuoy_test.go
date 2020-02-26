// +build k8srequired

package sonobuoy

import (
	"context"
	"testing"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"k8s.io/client-go/kubernetes"
)

func Test_Sonobuoy(t *testing.T) {
	err := sonobuoy.Test(context.Background())
	if err != nil {
		t.Fatalf("%#v", err)
	}
}

type Config struct {
	Logger    micrologger.Logger
	K8sClient kubernetes.Interface
	Provider  *Provider
}

type Sonobuoy struct {
	logger    micrologger.Logger
	k8sClient kubernetes.Interface
	provider  *Provider
}

func New(config Config) (*Sonobuoy, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.Provider == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Provider must not be empty", config)
	}
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}

	s := &Sonobuoy{
		logger:    config.Logger,
		k8sClient: config.K8sClient,
		provider:  config.Provider,
	}

	return s, nil
}

func (s *Sonobuoy) Test(ctx context.Context) error {
	return nil
}
