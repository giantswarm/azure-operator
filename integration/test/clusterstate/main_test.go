// +build k8srequired

package clusterstate

import (
	"testing"

	"github.com/giantswarm/e2etests/clusterstate"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v4/integration/env"
	"github.com/giantswarm/azure-operator/v4/integration/setup"
)

var (
	config           setup.Config
	clusterStateTest *clusterstate.ClusterState
)

func init() {
	var err error

	{
		config, err = setup.NewConfig()
		if err != nil {
			panic(microerror.JSON(err))
		}
	}

	var p *Provider
	{
		c := ProviderConfig{
			AzureClient: config.AzureClient,
			G8sClient:   config.Host.G8sClient(),
			Logger:      config.Logger,

			ClusterID: env.ClusterID(),
		}

		p, err = NewProvider(c)
		if err != nil {
			panic(microerror.JSON(err))
		}
	}

	{
		c := clusterstate.Config{
			LegacyFramework: config.Guest,
			Logger:          config.Logger,
			Provider:        p,
		}

		clusterStateTest, err = clusterstate.New(c)
		if err != nil {
			panic(microerror.JSON(err))
		}
	}
}

// TestMain allows us to have common setup and teardown steps that are run
// once for all the tests https://golang.org/pkg/testing/#hdr-Main.
func TestMain(m *testing.M) {
	setup.WrapTestMain(m, config)
}
