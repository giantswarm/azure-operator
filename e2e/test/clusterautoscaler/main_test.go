// +build k8srequired

package clusterautoscaler

import (
	"testing"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v5/e2e/setup"
)

var (
	config     setup.Config
	autoscaler *ClusterAutoscaler
)

func init() {
	var err error
	{
		config, err = setup.NewConfig()
		if err != nil {
			panic(microerror.JSON(err))
		}
	}

	{
		c := Config{
			Logger: config.Logger,
			Guest:  config.Guest,
		}

		autoscaler, err = New(c)
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
