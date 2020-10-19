package vmsscheck

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/Azure/go-autorest/autorest"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
)

const (
	// Provisioning States.
	provisioningStateFailed    = "Failed"
	provisioningStateSucceeded = "Succeeded"

	// Max number of HighCostGetVMScaleSet calls that can be made during a 30-minute period
	remainingCallsMax30m = 900

	// Max number of HighCostGetVMScaleSet calls that can be made during a 3-minute period
	remainingCallsMax3m = 190

	// Key used to extract remaining number of calls for 30 minutes from remainingCallsHeaderName
	remainingCallsHeaderKey30m = "Microsoft.Compute/HighCostGetVMScaleSet30Min"

	// Key used to extract remaining number of calls for 3 minutes from remainingCallsHeaderName
	remainingCallsHeaderKey3m = "Microsoft.Compute/HighCostGetVMScaleSet3Min"

	// Response header name that has info about remaining number of HighCostGetVMScaleSet calls.
	// Header example:
	// Microsoft.Compute/HighCostGetVMScaleSet3Min;107,Microsoft.Compute/HighCostGetVMScaleSet30Min;827
	remainingCallsHeaderName = "X-Ms-Ratelimit-Remaining-Resource"

	// If the number of remaining calls for 30min drops below this threshold, we do not proceed
	remainingCallsThreshold30m = remainingCallsMax30m * 0.5

	// If the number of remaining calls for 3min drops below this threshold, we do not proceed
	remainingCallsThreshold3m = remainingCallsMax3m * 0.5
)

// Find out provisioning state of all VMSS instances and return true if all are
// Succeeded.
func InstancesAreRunning(ctx context.Context, logger micrologger.Logger, virtualMachineScaleSetVMsClient *compute.VirtualMachineScaleSetVMsClient, resourceGroup string, vmssName string) (bool, error) {
	// Get a list of instances in the VMSS.
	iterator, err := virtualMachineScaleSetVMsClient.ListComplete(ctx, resourceGroup, vmssName, "", "", "")
	if err != nil {
		return false, microerror.Mask(err)
	}

	allSucceeded := true
	response := iterator.Response().Response

	for iterator.NotDone() {
		instance := iterator.Value()
		logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Instance %s has state %s", *instance.Name, *instance.ProvisioningState))

		switch *instance.ProvisioningState {
		case provisioningStateFailed:
			allSucceeded = false
		case provisioningStateSucceeded:
			// OK to continue.
		default:
			allSucceeded = false
		}

		if err := iterator.NextWithContext(ctx); err != nil {
			return false, microerror.Mask(err)
		}
	}

	if allSucceeded {
		// All instances are succeeded, let's check the VMSS rate is safe.
		// If current remaining API calls are less than the desired threshold, we don't proceed.
		rl3m, rl30m := rateLimitThresholdsFromResponse(response)
		if rl3m < remainingCallsThreshold3m || rl30m < remainingCallsThreshold30m {
			logger.LogCtx(ctx, "level", "warning", "message", fmt.Sprintf("The VMSS API remaining calls are not safe to continue (3m %d/%d, 30m %d/%d)", rl3m, remainingCallsMax3m, rl30m, remainingCallsMax30m)) // nolint: errcheck
			return false, vmssUnsafeError
		}
	}

	return allSucceeded, nil
}

func rateLimitThresholdsFromResponse(response autorest.Response) (int64, int64) {
	headers := response.Header[remainingCallsHeaderName]

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
			case remainingCallsHeaderKey3m:
				rl3m = val
			case remainingCallsHeaderKey30m:
				rl30m = val
			}
		}
	}

	return rl3m, rl30m
}
