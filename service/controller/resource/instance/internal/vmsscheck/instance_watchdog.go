package vmsscheck

import (
	"context"
	"sync"

	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/azure-operator/service/controller/resource/instance/internal/workerpool"
)

type Config struct {
	Logger micrologger.Logger

	NumWorkers int
}

type concurrentInstanceWatchdog struct {
	vmssGuards *sync.Map
	pool       *workerpool.Pool
}

func NewInstanceWatchdog(config Config) (InstanceWatchdog, error) {
	if config.Logger == nil {
	}

	wd := &concurrentInstanceWatchdog{
		vmssGuards: new(sync.Map),
		pool:       workerpool.New(config.NumWorkers, config.Logger),
	}

	return wd, nil
}

func (wd *concurrentInstanceWatchdog) GuardVMSS(ctx context.Context, resourceGroupName, vmssName string) {
	jobID := vmssGuardName(resourceGroupName, vmssName)
	job := &guardJob{
		id:            jobID,
		resourceGroup: resourceGroupName,
		vmss:          vmssName,

		onFinished: func() { wd.vmssGuards.Delete(jobID) },
	}

	_, exists := wd.vmssGuards.LoadOrStore(job.id, job)
	if exists {
		return
	}

	wd.pool.EnqueueJob(job)
}

func vmssGuardName(resourceGroupName, vmssGuardName string) string {
	return resourceGroupName + "/" + vmssGuardName
}
