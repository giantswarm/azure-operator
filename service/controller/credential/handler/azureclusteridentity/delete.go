package azureclusteridentity

import "context"

func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	r.logger.Debugf(ctx, "AzureClusterIdentity::EnsureDeleted")
	return nil
}
