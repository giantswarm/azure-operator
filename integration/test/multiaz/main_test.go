// +build k8srequired

package multiaz

import (
	"context"
	"testing"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/azure-operator/integration/env"
	"github.com/giantswarm/azure-operator/integration/setup"
)

var (
	config  setup.Config
	multiaz *MultiAZ
)

func init() {
	var err error
	{
		config, err = setup.NewConfig()
		if err != nil {
			panic(err.Error())
		}
	}

	var p *Provider
	{
		c := ProviderConfig{
			AzureClient: config.AzureClient,
			G8sClient:   config.Host.G8sClient(),
			Logger:      config.Logger,
			ClusterID:   env.ClusterID(),
		}

		p, err = NewProvider(c)
		if err != nil {
			panic(err.Error())
		}
	}

	{
		c := Config{
			Logger:   config.Logger,
			Provider: p,
		}

		multiaz, err = New(c)
		if err != nil {
			panic(err.Error())
		}
	}
}

// TestMain allows us to have common setup and teardown steps that are run
// once for all the tests https://golang.org/pkg/testing/#hdr-Main.
func TestMain(m *testing.M) {
	setup.WrapTestMain(m, config)
}

type Config struct {
	Logger   micrologger.Logger
	Provider *Provider
}

type MultiAZ struct {
	logger   micrologger.Logger
	provider *Provider
}

func New(config Config) (*MultiAZ, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.Provider == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Provider must not be empty", config)
	}

	s := &MultiAZ{
		logger:   config.Logger,
		provider: config.Provider,
	}

	return s, nil
}

func (s *MultiAZ) Test(ctx context.Context) error {
	azs, err := s.provider.GetClusterAZs(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	if len(azs) != 1 {
		return microerror.Mask(wrongAZzUsed)
	}
	if azs[0] != 1 {
		return microerror.Mask(wrongAZzUsed)
	}

	return nil
}
