package ipam

import (
	"context"
	"fmt"
	"net"

	"github.com/giantswarm/ipam"
	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/api/meta"

	"github.com/giantswarm/azure-operator/v8/pkg/locker"
)

// EnsureCreated allocates tenant cluster network segments. It gathers existing
// subnets from existing system resources like Vnets and Cluster CRs.
func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
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

	// 1/4 Check if a vnet/subnet is already allocated.
	{
		subnet, err := r.checker.Check(ctx, m.GetNamespace(), m.GetName())
		if err != nil {
			return microerror.Mask(err)
		}

		if subnet != nil {
			r.logger.Debugf(ctx, "%s already allocated", r.networkRangeType)
			r.logger.Debugf(ctx, "canceling resource")
			return nil
		}
	}

	// 2/4 Since we need to allocate a new vnet/subnet, first let's get all already allocated vnets/subnets.
	var allocatedNetworkRanges []net.IPNet
	{
		r.logger.Debugf(ctx, "finding allocated %ss", r.networkRangeType)

		allocatedNetworkRanges, err = r.collector.Collect(ctx, obj)
		if IsParentNetworkRangeStillNotKnown(err) {
			// We cancel IPAM reconciliation, which should be done in one of the next
			// reconciliation loops, as soon as the parent network range is allocated. See
			// IsParentNetworkRangeStillNotKnown function for more details.
			warningMessage := fmt.Sprintf(
				"parent network range from which the %s should be allocated is still not known, look for previous warnings for more details, skipping IPAM reconciliation",
				r.networkRangeType)
			r.logger.LogCtx(ctx, "level", "warning", "message", warningMessage)
			return nil
		} else if IsNotFound(err) {
			warningMessage := "resource not found, look for previous warnings for more details, skipping IPAM reconciliation"
			r.logger.LogCtx(ctx, "level", "warning", "message", warningMessage)
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}

		r.logger.Debugf(ctx, "found allocated %ss %#q", r.networkRangeType, allocatedNetworkRanges)
	}

	// 3/4 Now let when we know what vnets/subnets are allocated, let's get one that's available.
	var freeNetworkRange net.IPNet
	{
		r.logger.Debugf(ctx, "finding free %s", r.networkRangeType)

		parentNetworkRange, err := r.networkRangeGetter.GetParentNetworkRange(ctx, obj)
		if IsParentNetworkRangeStillNotKnown(err) {
			// We cancel IPAM reconciliation, which should be done in one of the next
			// reconciliation loops, as soon as the parent network range is allocated. See
			// IsParentNetworkRangeStillNotKnown function for more details.
			warningMessage := fmt.Sprintf(
				"parent network range from which the %s should be allocated is still not known, look for previous warnings for more details, skipping IPAM reconciliation",
				r.networkRangeType)
			r.logger.LogCtx(ctx, "level", "warning", "message", warningMessage)
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}
		requiredIPMask := r.networkRangeGetter.GetRequiredIPMask()

		freeNetworkRange, err = ipam.Free(parentNetworkRange, requiredIPMask, allocatedNetworkRanges)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.Debugf(ctx, "found free %s %#q", r.networkRangeType, freeNetworkRange)
	}

	// 4/4 And finally, let's save newly allocated network range (vnet range for cluster or subnet range node pool).
	{
		r.logger.Debugf(ctx, "allocating free %s %#q", r.networkRangeType, freeNetworkRange)

		err = r.persister.Persist(ctx, freeNetworkRange, m.GetNamespace(), m.GetName())
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.Debugf(ctx, "allocated free %s %#q", r.networkRangeType, freeNetworkRange)
	}

	return nil
}
