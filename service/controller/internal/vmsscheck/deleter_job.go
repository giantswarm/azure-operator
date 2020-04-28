package vmsscheck

import (
	"context"
	"fmt"
	"time"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
)

type deleterJob struct {
	context context.Context
	logger  micrologger.Logger

	id                    string
	resourceGroup         string
	vmss                  string
	nextExecutionTime     time.Time
	allInstancesSucceeded bool

	onFinished func()
}

func (gj *deleterJob) ID() string {
	return gj.id
}

func (gj *deleterJob) Run() error {
	// Still not the time to run the check
	if !time.Now().After(gj.nextExecutionTime) {
		return nil
	}

	var err error
	gj.allInstancesSucceeded, err = gj.deleteFailedInstances(gj.context, gj.resourceGroup, gj.vmss)
	if err != nil {
		return microerror.Mask(err)
	}

	gj.nextExecutionTime = time.Now().Add(10 * time.Second)
	return nil
}

func (gj *deleterJob) Finished() bool {
	// If any of the VMSS instances are in Failed state, return false here.
	if !gj.allInstancesSucceeded {
		return false
	}

	gj.onFinished()
	return true
}

// If any of the instances is not Succeeded, returns false.
// It deletes instances that are in "Failed" state.
func (gj *deleterJob) deleteFailedInstances(ctx context.Context, rg string, vmssName string) (bool, error) {
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

		gj.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Instance %s has state %s", *instance.Name, *instance.ProvisioningState))

		switch *instance.ProvisioningState {
		case provisioningStateFailed:
			// Reimage the instance.
			gj.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Deleting instance %s", *instance.Name))

			retries := 3
			for retries > 0 {
				_, err := c.Delete(ctx, rg, vmssName, *instance.InstanceID)
				if err != nil {
					gj.logger.LogCtx(ctx, "level", "error", "message", fmt.Sprintf("Error deleting instance %s: %s", *instance.Name, err.Error()))
					if retries == 0 {
						return false, microerror.Mask(err)
					}

					retries = retries - 1
					time.Sleep(5 * time.Second)
					continue
				}

				break
			}

			gj.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Deleted instance %s", *instance.Name))
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
