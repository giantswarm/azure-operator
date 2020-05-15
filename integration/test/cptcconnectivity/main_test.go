// +build k8srequired

package cptcconnectivity

import (
	"testing"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v4/integration/env"
	"github.com/giantswarm/azure-operator/v4/integration/setup"
)

var (
	config       setup.Config
	connectivity *Connectivity
)

func init() {
	var err error
	config, err = setup.NewConfig()
	if err != nil {
		panic(microerror.JSON(err))
	}

	{
		c := Config{
			Logger:    config.Logger,
			K8sClient: config.Host.K8sClient(),
			G8sClient: config.Host.G8sClient(),
			ClusterID: env.ClusterID(),
		}

		connectivity, err = New(c)
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
