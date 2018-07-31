package vnetpeering

import (
	"context"
)

// ApplyCreateChange is noop.
func (r *Resource) ApplyCreateChange(ctx context.Context, obj, createChange interface{}) error {
	return nil
}
