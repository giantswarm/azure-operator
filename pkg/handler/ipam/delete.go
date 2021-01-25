package ipam

import (
	"context"
	"net"

	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/api/meta"

	"github.com/giantswarm/azure-operator/v5/pkg/locker"
)

func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	var err error

	m, err := meta.Accessor(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	{
		r.logger.Debugf(ctx, "acquiring lock for IPAM")
		err := r.locker.Lock(ctx)
		if locker.IsAlreadyExists(err) {
			r.logger.Debugf(ctx, "lock for IPAM is already acquired")
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			r.logger.Debugf(ctx, "acquired lock for IPAM")
		}

		defer func() {
			r.logger.Debugf(ctx, "releasing lock for IPAM")
			err := r.locker.Unlock(ctx)
			if locker.IsNotFound(err) {
				r.logger.Debugf(ctx, "lock for IPAM is already released")
			} else if err != nil {
				r.logger.Errorf(ctx, err, "failed to release lock for IPAM")
			} else {
				r.logger.Debugf(ctx, "released lock for IPAM")
			}
		}()
	}

	// Check if subnet is still allocated.
	var subnet *net.IPNet
	{
		r.logger.Debugf(ctx, "finding if subnet is still allocated")

		subnet, err = r.checker.Check(ctx, m.GetNamespace(), m.GetName())
		if err != nil {
			return microerror.Mask(err)
		}

		if subnet == nil {
			r.logger.Debugf(ctx, "did not find allocated subnet")
			return nil
		}

		r.logger.Debugf(ctx, "found allocated subnet")
		r.logger.Debugf(ctx, "releasing allocated subnet")

		// Release allocated subnet.
		err = r.releaser.Release(ctx, *subnet, m.GetNamespace(), m.GetName())
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.Debugf(ctx, "released allocated subnet")
	}

	return nil
}
