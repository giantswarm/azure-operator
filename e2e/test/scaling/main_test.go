// +build k8srequired

package scaling

import (
	"context"
	"testing"

	releasev1alpha1 "github.com/giantswarm/apiextensions/v2/pkg/apis/release/v1alpha1"
	"github.com/giantswarm/e2etests/v2/scaling"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v4/e2e/env"
	"github.com/giantswarm/azure-operator/v4/e2e/setup"
)

var (
	config      setup.Config
	scalingTest *scaling.Scaling
)

func init() {
	ctx := context.Background()

	var err error
	var release *releasev1alpha1.Release
	{
		release, err = setup.CreateGSReleaseContainingOperatorVersion(ctx, config)
		if err != nil {
			panic(microerror.JSON(err))
		}
	}

	var config setup.Config
	{
		config, err = setup.NewConfig(release)
		if err != nil {
			panic(microerror.JSON(err))
		}
	}

	var p *Provider
	{
		c := ProviderConfig{
			GuestFramework: config.Guest,
			HostFramework:  config.Host,
			Logger:         config.Logger,

			ClusterID:  env.ClusterID(),
			CtrlClient: config.K8sClients.CtrlClient(),
		}

		p, err = NewProvider(c)
		if err != nil {
			panic(microerror.JSON(err))
		}
	}

	{
		c := scaling.Config{
			Logger:   config.Logger,
			Provider: p,
		}

		scalingTest, err = scaling.New(c)
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
