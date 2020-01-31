package minikube

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/giantswarm/backoff"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/e2e-harness/pkg/builder"
	"github.com/giantswarm/e2e-harness/pkg/harness"
)

type Minikube struct {
	logger   micrologger.Logger
	builder  builder.Builder
	imageTag string
}

func New(logger micrologger.Logger, builder builder.Builder, tag string) *Minikube {
	return &Minikube{
		logger:   logger,
		builder:  builder,
		imageTag: tag,
	}
}

// BuildImages is a Task that build the required images for both the main
// project and the e2e containers using the minikube docker environment.
func (m *Minikube) BuildImages(ctx context.Context) error {
	dir, err := os.Getwd()
	if err != nil {
		return microerror.Mask(err)
	}
	image := fmt.Sprintf("quay.io/giantswarm/%s", harness.GetProjectName())

	m.logger.Log("level", "info", "message", fmt.Sprintf("building image %q", image))

	o := func() error {
		err := m.builder.Build(ioutil.Discard, image, dir, m.imageTag, nil)
		if err != nil {
			return microerror.Mask(err)
		}
		return nil
	}
	b := backoff.NewExponential(backoff.ShortMaxWait, backoff.ShortMaxInterval)
	n := backoff.NewNotifier(m.logger, ctx)

	err = backoff.RetryNotify(o, b, n)
	if err != nil {
		return microerror.Mask(err)
	}

	m.logger.Log("level", "info", "message", fmt.Sprintf("built image %q", image))

	return nil
}
