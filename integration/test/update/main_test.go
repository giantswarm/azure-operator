// +build k8srequired

package update

import (
	"testing"
	"time"

	"github.com/giantswarm/e2etests/v2/update"

	"github.com/giantswarm/azure-operator/v4/integration/env"
	"github.com/giantswarm/azure-operator/v4/integration/setup"
)

var (
	config     setup.Config
	updateTest *update.Update
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
			G8sClient: config.Host.G8sClient(),
			Logger:    config.Logger,

			ClusterID: env.ClusterID(),
		}

		p, err = NewProvider(c)
		if err != nil {
			panic(err.Error())
		}
	}

	{
		c := update.Config{
			Logger:   config.Logger,
			Provider: p,

			MaxWait: 90 * time.Minute,
		}

		updateTest, err = update.New(c)
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
