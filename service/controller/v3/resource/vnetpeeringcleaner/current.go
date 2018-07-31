package vnetpeering

import (
	"context"
)

// GetCurrentState  is noop.
func (r *Resource) GetCurrentState(ctx context.Context, obj interface{}) (interface{}, error) {
	return nil, nil
}
