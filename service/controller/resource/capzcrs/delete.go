package capzcrs

import (
	"context"
	"fmt"
)

func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	fmt.Println("====> delete event on capzcrs handler")
	// Once cluster has been migrated to node pools, CAPI & CAPZ CRs are
	// deleted by api and AzureConfig is deleted by AzureCluster reconciliation
	// so nothing to do here.
	return nil
}
