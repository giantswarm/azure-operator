package instance

import (
	"context"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return microerror.Mask(err)
	}
	err = r.ClientFactory.RemoveAllClients(cr)
	if err != nil {
		return microerror.Mask(err)
	}
	return nil
}
