package setup

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/integration/env"
)

const (
	azureResourceValuesFile = "/tmp/azure-operator-values.yaml"
)

// WrapTestMain setup and teardown e2e testing environment.
func WrapTestMain(m *testing.M, c Config) {
	var r int

	ctx := context.Background()

	err := Setup(ctx, c)
	if err != nil {
		log.Printf("%#v\n", err)
		r = 1
	} else {
		r = m.Run()
	}

	if env.KeepResources() != "true" {
		Teardown(c)
	}

	os.Exit(r)
}

// Setup e2e testing environment.
func Setup(ctx context.Context, c Config) error {
	var err error

	err = common(ctx, c)
	if err != nil {
		return microerror.Mask(err)
	}

	err = provider(ctx, c)
	if err != nil {
		return microerror.Mask(err)
	}

	err = c.Guest.Setup(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	// Uncomment the bastion code to deploy a VM with a public IP address so you can SSH into the cluster nodes.
	//err = bastion(ctx, c)
	//if err != nil {
	//	return microerror.Mask(err)
	//}

	return nil
}
