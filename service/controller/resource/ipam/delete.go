package ipam

import (
	"context"
	"fmt"
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
		r.logger.LogCtx(ctx, "level", "debug", "message", "acquiring lock for IPAM")
		err := r.locker.Lock(ctx)
		if locker.IsAlreadyExists(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "lock for IPAM is already acquired")
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", "acquired lock for IPAM")
		}

		defer func() {
			r.logger.LogCtx(ctx, "level", "debug", "message", "releasing lock for IPAM")
			err := r.locker.Unlock(ctx)
			if locker.IsNotFound(err) {
				r.logger.LogCtx(ctx, "level", "debug", "message", "lock for IPAM is already released")
			} else if err != nil {
				r.logger.LogCtx(ctx, "level", "error", "message", "failed to release lock for IPAM", "stack", fmt.Sprintf("%#v", err))
			} else {
				r.logger.LogCtx(ctx, "level", "debug", "message", "released lock for IPAM")
			}
		}()
	}

	// Check if subnet is still allocated.
	var subnet *net.IPNet
	{
		r.logger.LogCtx(ctx, "level", "debug", "message", "finding if subnet is still allocated")

		subnet, err = r.checker.Check(ctx, m.GetNamespace(), m.GetName())
		if err != nil {
			return microerror.Mask(err)
		}

		if subnet == nil {
			r.logger.LogCtx(ctx, "level", "debug", "message", "did not find allocated subnet")
			return nil
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", "found allocated subnet")
		r.logger.LogCtx(ctx, "level", "debug", "message", "releasing allocated subnet")

		// Release allocated subnet.
		err = r.releaser.Release(ctx, *subnet, m.GetNamespace(), m.GetName())
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", "released allocated subnet")
	}

	return nil
}
