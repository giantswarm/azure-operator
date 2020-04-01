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

	"github.com/giantswarm/azure-operator/service/controller/controllercontext"
)

// Find out provisioning state of all VMSS instances and return true if all are
// Succeeded.
func InstancesAreRunning(ctx context.Context, logger micrologger.Logger, rg string, vmssName string) (bool, error) {
	c, err := getVMsClient(ctx)
	if err != nil {
		return false, microerror.Mask(err)
	}

	// Get a list of instances in the VMSS.
	iterator, err := c.ListComplete(ctx, rg, vmssName, "", "", "")
	if err != nil {
		return false, microerror.Mask(err)
	}

	allSucceeded := true

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

		if err := iterator.Next(); err != nil {
			return false, microerror.Mask(err)
		}
	}

	return allSucceeded, nil
}

func getVMsClient(ctx context.Context) (*compute.VirtualMachineScaleSetVMsClient, error) {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return cc.AzureClientSet.VirtualMachineScaleSetVMsClient, nil
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
