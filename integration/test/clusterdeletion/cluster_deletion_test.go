// +build k8srequired

package clusterdeletion

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/Azure/go-autorest/autorest"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
)

func Test_ClusterDeletion(t *testing.T) {
	err := deletecluster.Test(context.Background())
	if err != nil {
		t.Fatalf("%#v", err)
	}
}

type Config struct {
	Logger   micrologger.Logger
	Provider *Provider
}

type ClusterDeletion struct {
	logger   micrologger.Logger
	provider *Provider
}

func New(config Config) (*ClusterDeletion, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.Provider == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Provider must not be empty", config)
	}

	s := &ClusterDeletion{
		logger:   config.Logger,
		provider: config.Provider,
	}

	return s, nil
}

func (s *ClusterDeletion) Test(ctx context.Context) error {
	s.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensuring deletion of Azure Resource Group %#q", s.provider.clusterID))
	o := func() error {
		_, err := s.provider.azureClient.ResourceGroupsClient.Get(ctx, s.provider.clusterID)
		if err != nil {
			reqError, ok := err.(autorest.DetailedError)
			if ok {
				if reqError.StatusCode == http.StatusNotFound {
					return nil
				}
			}

			return microerror.Mask(err)
		}

		return microerror.Maskf(executionFailedError, "The resource group still exists")
	}
	b := backoff.NewExponential(60*time.Minute, backoff.LongMaxInterval)
	n := backoff.NewNotifier(s.logger, ctx)
	err := backoff.RetryNotify(o, b, n)
	if err != nil {
		s.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("did not ensure deletion of Azure Resource Group %#q", s.provider.clusterID))
		return microerror.Mask(err)
	}

	return nil
}
