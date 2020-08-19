package ipam

import (
	"context"
	"fmt"
	"net"

	"github.com/giantswarm/ipam"
	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/api/meta"

	"github.com/giantswarm/azure-operator/v4/pkg/locker"
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

	// 1/4 Check if a vnet/subnet is already allocated.
	{
		proceed, err := r.checker.Check(ctx, m.GetNamespace(), m.GetName())
		if err != nil {
			return microerror.Mask(err)
		}

		if !proceed {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("%s already allocated", r.networkRangeType))
			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
			return nil
		}
	}

	// 2/4 Since we need to allocate a new vnet/subnet, first let's get all already allocated vnets/subnets.
	var allocatedNetworkRanges []net.IPNet
	{
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("finding allocated %ss", r.networkRangeType))

		allocatedNetworkRanges, err = r.collector.Collect(ctx, obj)
		if IsParentNetworkRangeStillNotKnown(err) {
			// This can happen in node pools IPAM reconciliation, during subnet allocation, when
			// AzureCluster.Spec.NetworkSpec.Vnet.CidrBlock is still not set, because VNet for the
			// tenant cluster is still not allocated (e.g. when cluster is still being created). In
			// this case we cancel IPAM reconciliation, which will be done as soon as the VNet is
			// allocated and set in the AzureCluster CR.
			warningMessage := fmt.Sprintf(
				"parent network range from which the %s should be allocated is still not known, look for previous warnings for more details",
				r.networkRangeType)
			r.logger.LogCtx(ctx, "level", "warning", "message", warningMessage)
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found allocated %ss %#q", r.networkRangeType, allocatedNetworkRanges))
	}

	// 3/4 Now let when we know what vnets/subnets are allocated, let's get one that's available.
	var freeNetworkRange net.IPNet
	{
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("finding free %s", r.networkRangeType))

		parentNetworkRange, err := r.networkRangeGetter.GetParentNetworkRange(ctx, obj)
		if IsParentNetworkRangeStillNotKnown(err) {
			// This can happen in node pools IPAM reconciliation, during subnet allocation, when
			// AzureCluster.Spec.NetworkSpec.Vnet.CidrBlock is still not set, because VNet for the
			// tenant cluster is still not allocated (e.g. when cluster is still being created). In
			// this case we cancel IPAM reconciliation, which will be done as soon as the VNet is
			// allocated and set in the AzureCluster CR.
			warningMessage := fmt.Sprintf(
				"parent network range from which the %s should be allocated is still not known, look for previous warnings for more details",
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

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found free %s %#q", r.networkRangeType, freeNetworkRange))
	}

	// 4/4 And finally, let's save newly allocated network range (vnet range for cluster or subnet range node pool).
	{
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("allocating free %s %#q", r.networkRangeType, freeNetworkRange))

		err = r.persister.Persist(ctx, freeNetworkRange, m.GetNamespace(), m.GetName())
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("allocated free %s %#q", r.networkRangeType, freeNetworkRange))
	}

	return nil
}
