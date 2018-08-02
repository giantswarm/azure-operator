package vnetpeeringcleaner

import (
	"context"
)

// ApplyCreateChange is noop.
func (r *Resource) ApplyCreateChange(ctx context.Context, obj, createChange interface{}) error {
	r.logger.LogCtx(ctx, "level", "debug", "message", "ApplyCreateChange")
	return nil
}
