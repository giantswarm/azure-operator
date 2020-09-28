// +build k8srequired

package nodepool

import (
	"context"
	"testing"

	releasev1alpha1 "github.com/giantswarm/apiextensions/v2/pkg/apis/release/v1alpha1"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v4/e2e/env"
	"github.com/giantswarm/azure-operator/v4/e2e/setup"
)

var (
	config   setup.Config
	nodepool *Nodepool
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
