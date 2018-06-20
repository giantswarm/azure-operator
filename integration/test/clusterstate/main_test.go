// +build k8srequired

package clusterstate

import (
	"testing"

	"github.com/giantswarm/e2e-harness/pkg/framework"
	e2eclient "github.com/giantswarm/e2eclients/azure"
	"github.com/giantswarm/e2etests/clusterstate"
	"github.com/giantswarm/e2etests/clusterstate/provider"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/integration/env"
	"github.com/giantswarm/azure-operator/integration/setup"
)

var (
	c  *client.AzureClientSet
	cs *clusterstate.ClusterState
	g  *framework.Guest
	h  *framework.Host
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
		}

		g, err = framework.NewGuest(c)
		if err != nil {
			panic(err.Error())
		}
	}

	{
		c := framework.HostConfig{}

		h, err = framework.NewHost(c)
		if err != nil {
			panic(err.Error())
		}
	}

	var p *provider.Azure
	{
		ac, err := e2eclient.NewClient()
		if err != nil {
			panic(err.Error())
		}

		c := provider.AzureConfig{
			AzureClient:    ac,
			GuestFramework: g,
			HostFramework:  h,
			Logger:         logger,

			ClusterID: env.ClusterID(),
		}

		p, err = provider.NewAzure(c)
		if err != nil {
			panic(err.Error())
		}
	}

	{
		c := clusterstate.Config{
			Logger:   logger,
			Provider: p,
		}

		cs, err = clusterstate.New(c)
		if err != nil {
			panic(err.Error())
		}
	}

	{
		config := client.AzureClientSetConfig{
			Logger:         logger,
			ClientID:       env.AzureClientID(),
			ClientSecret:   env.AzureClientSecret(),
			SubscriptionID: env.AzureSubscriptionID(),
			TenantID:       env.AzureTenantID(),
		}

		c, err = client.NewAzureClientSet(config)
		if err != nil {
			panic(err.Error())
		}
	}
}

// TestMain allows us to have common setup and teardown steps that are run
// once for all the tests https://golang.org/pkg/testing/#hdr-Main.
func TestMain(m *testing.M) {
	setup.WrapTestMain(c, g, h, m)
}
