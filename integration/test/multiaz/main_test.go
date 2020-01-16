// +build k8srequired

package multiaz

import (
	"context"
	"fmt"
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
	s.logger.LogCtx(ctx, "level", "debug", "message", "getting current availability zones")
	vmss, err := s.provider.azureClient.VirtualMachineScaleSetsClient.Get(ctx, "s.provider.clusterID", fmt.Sprintf("%s-%s", s.provider.clusterID, "worker"))
	if err != nil {
		return microerror.Mask(err)
	}
	s.logger.LogCtx(ctx, "level", "debug", "message", "found availability zones", "azs", *vmss.Zones)

	if len(*vmss.Zones) != 1 {
		return microerror.Mask(wrongAzsUsed)
	}
	if (*vmss.Zones)[0] != "1" {
		return microerror.Mask(wrongAzsUsed)
	}

	return nil
}
