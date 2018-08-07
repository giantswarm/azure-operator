package vnetpeeringcleaner

import (
	"context"
)

// EnsureDeleted is noop.
func (r Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	return nil
}
