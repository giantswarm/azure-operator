// +build k8srequired

package hello

import (
	"os"
	"testing"

	"github.com/giantswarm/e2e-harness/pkg/framework"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/integration/setup"
)

var (
	g *framework.Guest
	h *framework.Host
	c *client.AzureClientSet
)

// TestMain allows us to have common setup and teardown steps that are run
// once for all the tests https://golang.org/pkg/testing/#hdr-Main.
func TestMain(m *testing.M) {
	var err error

	g, err = framework.NewGuest()
	if err != nil {
		panic(err.Error())
	}
	h, err = framework.NewHost(framework.HostConfig{})
	if err != nil {
		panic(err.Error())
	}

	var newLogger micrologger.Logger
	{
		loggerConfig := micrologger.Config{
			IOWriter: os.Stdout,
		}
		newLogger, err = micrologger.New(loggerConfig)
		if err != nil {
			panic(err.Error())
		}
	}

	var azureConfig = client.AzureConfig{
		Logger:         newLogger,
		ClientID:       os.Getenv("AZURE_CLIENTID"),
		ClientSecret:   os.Getenv("AZURE_CLIENTSECRET"),
		SubscriptionID: os.Getenv("AZURE_SUBSCRIPTIONID"),
		TenantID:       os.Getenv("AZURE_TENANTID"),
	}
	c, err := client.NewAzureClientSet(azureConfig)
	if err != nil {
		panic(err.Error())
	}

	setup.WrapTestMain(c, g, h, m)
}
