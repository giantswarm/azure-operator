package instance

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest"
	"github.com/giantswarm/microerror"
)

const (
	vmssVMListHeaderName = "X-Ms-Ratelimit-Remaining-Resource"
	headerKey3m          = "Microsoft.Compute/HighCostGetVMScaleSet3Min"
	headerKey30m         = "Microsoft.Compute/HighCostGetVMScaleSet30Min"
	max3m                = 190
	max30m               = 900
	threshold3m          = max3m * 0.5
	threshold30m         = max30m * 0.5
)

func checkVMSSApiRateLimitThresholds(response autorest.Response) (int64, int64) {
	headers := response.Header[vmssVMListHeaderName]

	rl3m := int64(-1)
	rl30m := int64(-1)

	for _, l := range headers {
		// Limits are a single comma separated string.
		tokens := strings.SplitN(l, ",", -1)
		for _, t := range tokens {
			// Each limit's name and value are separated by a semicolon.
			kv := strings.SplitN(t, ";", 2)
			if len(kv) != 2 {
				// We expect exactly two tokens, otherwise we ignore this header.
				continue
			}

			// The second token must be a number, otherwise we ignore this header.
			val, err := strconv.ParseInt(kv[1], 10, 32)
			if err != nil {
				continue
			}

			switch kv[0] {
			case headerKey3m:
				rl3m = val
			case headerKey30m:
				rl30m = val
			}
		}
	}

	return rl3m, rl30m
}

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

	// Check for rate limit. If current remaining API calls are less than the desider threshold, we don't proceed.
	rl3m, rl30m := checkVMSSApiRateLimitThresholds(iterator.Response().Response)
	if rl3m < threshold3m || rl30m < threshold30m {
		r.logger.LogCtx(ctx, "level", "warmomg", "message", fmt.Sprintf("The VMSS API remaining calls are not safe to continue (3m %d/%d, 30m %d/%d)", rl3m, max3m, rl30m, max30m))
		return false, nil
	}

	allSucceeded := true

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
			allSucceeded = false
		case ProvisioningStateSucceeded:
			// OK to continue.
		default:
			// Just wait.
			allSucceeded = false
		}

		if err := iterator.Next(); err != nil {
			return false, microerror.Mask(err)
		}
	}

	return allSucceeded, nil
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
