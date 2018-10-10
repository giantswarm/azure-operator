// +build k8srequired

package update

import (
	"testing"
	"time"

	"github.com/giantswarm/e2e-harness/pkg/framework"
	"github.com/giantswarm/e2etests/update"
	"github.com/giantswarm/e2etests/update/provider"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/azure-operator/integration/env"
	"github.com/giantswarm/azure-operator/integration/setup"
)

var (
	updateTest *update.Update
	g          *framework.Guest
	h          *framework.Host
)

func init() {
	var err error

	var logger micrologger.Logger
	{
		c := micrologger.Config{}

		logger, err = micrologger.New(c)
		if err != nil {
			panic(err.Error())
		}
	}

	{
		c := framework.GuestConfig{
			Logger: logger,

			ClusterID:    env.ClusterID(),
			CommonDomain: env.CommonDomain(),
		}

		g, err = framework.NewGuest(c)
		if err != nil {
			panic(err.Error())
		}
	}

	{
		c := framework.HostConfig{
			Logger: logger,

			ClusterID:  env.ClusterID(),
			VaultToken: env.VaultToken(),
		}

		h, err = framework.NewHost(c)
		if err != nil {
			panic(err.Error())
		}
	}

	var p *provider.Azure
	{
		c := provider.AzureConfig{
			HostFramework: h,
			Logger:        logger,

			ClusterID:   env.ClusterID(),
			GithubToken: env.GithubToken(),
		}

		p, err = provider.NewAzure(c)
		if err != nil {
			panic(err.Error())
		}
	}

	{
		c := update.Config{
			Logger:   logger,
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
	setup.WrapTestMain(g, h, m)
}
