package vmsscheck

import (
	"context"

	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/azure-operator/service/controller/resource/instance/internal/workerpool"
)

type Config struct {
	Logger micrologger.Logger

	NumWorkers int
}

type concurrentInstanceWatchdog struct {
	vmssGuards map[string]*guardJob
	pool       *workerpool.Pool
}

func NewInstanceWatchdog(config Config) (InstanceWatchdog, error) {
	if config.Logger == nil {
	}

	wd := &concurrentInstanceWatchdog{
		pool: workerpool.New(config.NumWorkers, config.Logger),
	}

	return wd, nil
}

func (wd *concurrentInstanceWatchdog) GuardVMSS(ctx context.Context, resourceGroupName, vmssName string) {
	guardName := vmssGuardName(resourceGroupName, vmssName)
	_, exists := wd.vmssGuards[guardName]
	if exists {
		return
	}

	gj := &guardJob{
		id: guardName,

		resourceGroup: resourceGroupName,
		vmss:          vmssName,
	}

	wd.pool.EnqueueJob(gj)
}

func vmssGuardName(resourceGroupName, vmssGuardName string) string {
	return resourceGroupName + "/" + vmssGuardName
}
