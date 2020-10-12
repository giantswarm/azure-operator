// +build k8srequired

package nodepool

import (
	"testing"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v5/e2e/env"
	"github.com/giantswarm/azure-operator/v5/e2e/setup"
)

var (
	config   setup.Config
	nodepool *Nodepool
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
			CtrlClient: config.K8sClients.CtrlClient(),
			Logger:     config.Logger,
		}

		p, err = NewProvider(c)
		if err != nil {
			panic(microerror.JSON(err))
		}
	}

	{
		c := Config{
			ClusterID:  env.ClusterID(),
			CtrlClient: config.K8sClients.CtrlClient(),
			Logger:     config.Logger,
			NodePoolID: env.NodePoolID(),
			Provider:   p,
			Guest:      config.Guest,
		}

		nodepool, err = New(c)
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
