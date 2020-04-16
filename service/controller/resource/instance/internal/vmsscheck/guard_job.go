package vmsscheck

import (
	"context"
	"fmt"
	"time"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
)

const (
	provisioningStateFailed    = "Failed"
	provisioningStateSucceeded = "Succeeded"

	// Key used to extract remaining number of calls for 30 minutes from remainingCallsHeaderName
	remainingCallsHeaderKey30m = "Microsoft.Compute/HighCostGetVMScaleSet30Min"

	// Key used to extract remaining number of calls for 3 minutes from remainingCallsHeaderName
	remainingCallsHeaderKey3m = "Microsoft.Compute/HighCostGetVMScaleSet3Min"

	// Response header name that has info about remaining number of HighCostGetVMScaleSet calls.
	// Header example:
	// Microsoft.Compute/HighCostGetVMScaleSet3Min;107,Microsoft.Compute/HighCostGetVMScaleSet30Min;827
	remainingCallsHeaderName = "X-Ms-Ratelimit-Remaining-Resource"

	// Max number of HighCostGetVMScaleSet calls that can be made during a 30-minute period
	remainingCallsMax30m = 900

	// Max number of HighCostGetVMScaleSet calls that can be made during a 3-minute period
	remainingCallsMax3m = 190

	// If the number of remaining calls for 30min drops below this threshold, we do not proceed
	remainingCallsThreshold30m = remainingCallsMax30m * 0.5

	// If the number of remaining calls for 3min drops below this threshold, we do not proceed
	remainingCallsThreshold3m = remainingCallsMax3m * 0.5
)

type guardJob struct {
	context context.Context
	logger  micrologger.Logger

	id                    string
	resourceGroup         string
	vmss                  string
	nextExecutionTime     time.Time
	allInstancesSucceeded bool

	onFinished func()
}

func (gj *guardJob) ID() string {
	return gj.id
}

func (gj *guardJob) Run() error {
	// Still not the time to run the check
	if !time.Now().After(gj.nextExecutionTime) {
		return nil
	}

	var err error
	gj.allInstancesSucceeded, err = gj.reimageFailedInstances(gj.context, gj.resourceGroup, gj.vmss)
	if err != nil {
		return microerror.Mask(err)
	}

	gj.nextExecutionTime = time.Now().Add(10 * time.Second)
	return nil
}

func (gj *guardJob) Finished() bool {
	// If any of the VMSS instances are in Failed state, return false here.
	if !gj.allInstancesSucceeded {
		return false
	}

	gj.onFinished()
	return true
}

// If any of the instances is not Succeeded, returns false.
// It reimages instances that are in "Failed" state.
func (gj *guardJob) reimageFailedInstances(ctx context.Context, rg string, vmssName string) (bool, error) {
	c, err := getVMsClient(ctx)
	if err != nil {
		return false, microerror.Mask(err)
	}

	// Get a list of instances in the VMSS.
	iterator, err := c.ListComplete(ctx, rg, vmssName, "", "", "")
	if err != nil {
		return false, microerror.Mask(err)
	}

	// Check for rate limit. If current remaining API calls are less than the desired threshold, we don't proceed.
	rl3m, rl30m := rateLimitThresholdsFromResponse(iterator.Response().Response)
	if rl3m < remainingCallsThreshold3m || rl30m < remainingCallsThreshold30m {
		gj.logger.LogCtx(ctx, "level", "warmomg", "message", fmt.Sprintf("The VMSS API remaining calls are not safe to continue (3m %d/%d, 30m %d/%d)", rl3m, remainingCallsMax3m, rl30m, remainingCallsMax30m)) // nolint: errcheck
		return false, nil
	}

	allSucceeded := true

	for iterator.NotDone() {
		instance := iterator.Value()

		gj.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Instance %s has state %s", *instance.Name, *instance.ProvisioningState)) // nolint: errcheck

		switch *instance.ProvisioningState {
		case provisioningStateFailed:
			// Reimage the instance.
			gj.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Reimaging instance %s", *instance.Name)) // nolint: errcheck

			retries := 3
			for retries > 0 {
				_, err := c.Reimage(ctx, rg, vmssName, *instance.InstanceID, nil)
				if err != nil {
					gj.logger.LogCtx(ctx, "level", "error", "message", fmt.Sprintf("Error reimaging instance %s: %s", *instance.Name, err.Error())) // nolint: errcheck
					if retries == 0 {
						return false, microerror.Mask(err)
					}

					retries = retries - 1
					time.Sleep(5 * time.Second)
					continue
				}

				break
			}

			gj.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Reimaged instance %s", *instance.Name)) // nolint: errcheck
			allSucceeded = false
		case provisioningStateSucceeded:
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
