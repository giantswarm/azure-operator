package blobobject

import (
	"context"

	"github.com/giantswarm/operatorkit/controller"
)

// ApplyDeleteChange not in use as blobobject deleted
// with container delete.
func (r *Resource) ApplyDeleteChange(ctx context.Context, obj, change interface{}) error {
	return nil
}

// NewDeletePatch is not in use as blobobject deleted
// with container delete.
func (r *Resource) NewDeletePatch(ctx context.Context, obj, currentState, desiredState interface{}) (*controller.Patch, error) {
	return nil, nil
}
