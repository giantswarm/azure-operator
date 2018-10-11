// +build k8srequired

package clusterstate

import (
	"testing"

	"github.com/giantswarm/e2etests/clusterstate"
	"github.com/giantswarm/e2etests/clusterstate/provider"

	"github.com/giantswarm/azure-operator/integration/env"
	"github.com/giantswarm/azure-operator/integration/setup"
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
			panic(err.Error())
		}
	}

	var p *provider.Azure
	{
		c := provider.AzureConfig{
			AzureClient:   config.AzureClient,
			HostFramework: config.Host,
			Logger:        config.Logger,

			ClusterID: env.ClusterID(),
		}

		p, err = provider.NewAzure(c)
		if err != nil {
			panic(err.Error())
		}
	}

	{
		c := clusterstate.Config{
			GuestFramework: config.Guest,
			Logger:         config.Logger,
			Provider:       p,
		}

		clusterStateTest, err = clusterstate.New(c)
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
