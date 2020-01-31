package tasks

import (
	"context"

	"github.com/giantswarm/microerror"
)

func Run(ctx context.Context, tasks []Task) error {
	var err error
	for _, task := range tasks {
		err = task(ctx)
		if err != nil {
			return microerror.Mask(err)
		}
	}
	return nil
}
