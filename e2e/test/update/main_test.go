// +build k8srequired

package update

import (
	"context"
	"testing"
	"time"

	releasev1alpha1 "github.com/giantswarm/apiextensions/v2/pkg/apis/release/v1alpha1"
	"github.com/giantswarm/e2etests/v2/update"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v4/e2e/env"
	"github.com/giantswarm/azure-operator/v4/e2e/setup"
)

var (
	config     setup.Config
	updateTest *update.Update
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
