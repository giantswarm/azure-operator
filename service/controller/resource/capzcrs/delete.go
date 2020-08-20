package capzcrs

import (
	"context"
)

func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	r.logger.LogCtx(ctx, "level", "debug", "message", "delete event on capzcrs handler")
	// Once cluster has been migrated to node pools, CAPI & CAPZ CRs are
	// deleted by api and AzureConfig is deleted by AzureCluster reconciliation
	// so nothing to do here.
	return nil
}
