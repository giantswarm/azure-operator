package cloudconfig

import (
	"context"

	"github.com/giantswarm/operatorkit/v2/pkg/resource/crud"
)

// ApplyDeleteChange not in use as blobobject deleted
// with container delete.
func (r *Resource) ApplyDeleteChange(ctx context.Context, obj, change interface{}) error {
	return nil
}

// NewDeletePatch is not in use as blobobject deleted
// with container delete.
func (r *Resource) NewDeletePatch(ctx context.Context, obj, currentState, desiredState interface{}) (*crud.Patch, error) {
	return nil, nil
}
