// +build k8srequired

package multiaz

import (
	"context"
	"testing"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
)

func Test_AZ(t *testing.T) {

	err := multiaz.Test(context.Background())
	if err != nil {
		t.Fatalf("%#v", err)
	}
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
	zones, err := s.provider.GetClusterAZs(ctx)
	if err != nil {
		return microerror.Mask(err)
	}
	s.logger.LogCtx(ctx, "level", "debug", "message", "found availability zones", "azs", zones)

	if len(zones) != 1 {
		return microerror.Maskf(executionFailedError, "The amount of AZ's used is not correct. Expected %d zones, got %d zones", 1, len(zones))
	}
	if zones[0] != "1" {
		return microerror.Maskf(executionFailedError, "The AZ used is not correct. Expected %s, got %s", "1", zones[0])
	}

	return nil
}
