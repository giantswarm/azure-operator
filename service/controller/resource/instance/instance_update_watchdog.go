package instance

import (
	"context"
	"fmt"
	"time"

	"github.com/giantswarm/microerror"
)

// If any of the instances is not Succeeded, returns false.
// It reimages instances that are in "Failed" state.
func (r *Resource) ensureWorkerInstancesAreAllRunning(ctx context.Context, rg string, vmssName string) (bool, error) {
	c, err := r.getVMsClient(ctx)
	if err != nil {
		return false, microerror.Mask(err)
	}

	// Get a list of instances in the VMSS.
	iterator, err := c.ListComplete(ctx, rg, vmssName, "", "", "")
	if err != nil {
		return false, microerror.Mask(err)
	}

	// TODO check for rate limit.

	for iterator.NotDone() {
		instance := iterator.Value()

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Instance %s has state %s", *instance.Name, *instance.ProvisioningState))

		switch *instance.ProvisioningState {
		case ProvisioningStateFailed:
			// Reimage the instance.
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Reimaging instance %s", *instance.Name))

			retries := 3
			for retries > 0 {
				_, err := c.Reimage(ctx, rg, vmssName, *instance.InstanceID, nil)
				if err != nil {
					r.logger.LogCtx(ctx, "level", "error", "message", fmt.Sprintf("Error reimaging instance %s: %s", *instance.Name, err.Error()))
					if retries == 0 {
						return false, microerror.Mask(err)
					}

					retries = retries - 1
					time.Sleep(5 * time.Second)
				}

				break
			}

			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Reimaged instance %s", *instance.Name))
			return false, nil
		case ProvisioningStateSucceeded:
			// OK to continue.
		default:
			// Just wait.
			return false, nil
		}

		if err := iterator.Next(); err != nil {
			return false, microerror.Mask(err)
		}
	}

	return true, nil
}

func (r *Resource) startInstanceWatchdog(ctx context.Context, rg string, vmssName string) error {
	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("[WATCHDOG] Starting watchdog for VMSS %s", vmssName))

	// Give some time to Azure for beginning the update.
	time.Sleep(60 * time.Second)

	for {
		success, err := r.ensureWorkerInstancesAreAllRunning(ctx, rg, vmssName)
		if err != nil {
			return microerror.Mask(err)
		}

		if success {
			break
		}

		time.Sleep(10 * time.Second)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("[WATCHDOG] Stopping watchdog for VMSS %s", vmssName))
	return nil
}
