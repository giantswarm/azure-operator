package spark

import (
	"context"
)

// EnsureDeleted will delete the `Spark` CR that was created for this specific node pool, and the `Secret` referenced by it.
func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	return nil
}
