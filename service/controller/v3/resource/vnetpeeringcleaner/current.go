package vnetpeeringcleaner

import (
	"context"
)

// GetCurrentState  is noop.
func (r *Resource) GetCurrentState(ctx context.Context, obj interface{}) (interface{}, error) {
	r.logger.LogCtx(ctx, "level", "debug", "message", "GetCurrentState")
	return nil, nil
}
