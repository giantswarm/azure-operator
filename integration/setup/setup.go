// +build k8srequired

package setup

import (
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

	err := Setup(c)
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
func Setup(c Config) error {
	var err error

	err = common(c)
	if err != nil {
		return microerror.Mask(err)
	}

	err = provider(c)
	if err != nil {
		return microerror.Mask(err)
	}

	err = c.Guest.Setup()
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
