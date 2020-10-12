package vmsscheck

import (
	"context"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/azure-operator/v5/service/controller/internal/workerpool"
)

type Config struct {
	Logger micrologger.Logger

	NumWorkers int
}

type concurrentInstanceWatchdog struct {
	vmssGuards *sync.Map
	pool       *workerpool.Pool
	logger     micrologger.Logger
}

func NewInstanceWatchdog(config Config) (InstanceWatchdog, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	wd := &concurrentInstanceWatchdog{
		vmssGuards: new(sync.Map),
		pool:       workerpool.New(config.NumWorkers, config.Logger),
		logger:     config.Logger,
	}

	return wd, nil
}

func (wd *concurrentInstanceWatchdog) GuardVMSS(ctx context.Context, virtualMachineScaleSetVMsClient *compute.VirtualMachineScaleSetVMsClient, resourceGroupName, vmssName string) {
	jobID := vmssGuardName(resourceGroupName, vmssName, "reimage")
	job := &guardJob{
		id:                              jobID,
		resourceGroup:                   resourceGroupName,
		vmss:                            vmssName,
		nextExecutionTime:               time.Now().Add(60 * time.Second),
		context:                         ctx,
		logger:                          wd.logger,
		virtualMachineScaleSetVMsClient: virtualMachineScaleSetVMsClient,

		onFinished: func() { wd.vmssGuards.Delete(jobID) },
	}

	_, exists := wd.vmssGuards.LoadOrStore(job.id, job)
	if exists {
		return
	}

	wd.pool.EnqueueJob(job)
}

func (wd *concurrentInstanceWatchdog) DeleteFailedVMSS(ctx context.Context, virtualMachineScaleSetVMsClient *compute.VirtualMachineScaleSetVMsClient, resourceGroupName, vmssName string) {
	jobID := vmssGuardName(resourceGroupName, vmssName, "delete")
	job := &deleterJob{
		id:                              jobID,
		resourceGroup:                   resourceGroupName,
		vmss:                            vmssName,
		nextExecutionTime:               time.Now().Add(60 * time.Second),
		context:                         ctx,
		logger:                          wd.logger,
		virtualMachineScaleSetVMsClient: virtualMachineScaleSetVMsClient,

		onFinished: func() { wd.vmssGuards.Delete(jobID) },
	}

	_, exists := wd.vmssGuards.LoadOrStore(job.id, job)
	if exists {
		return
	}

	wd.pool.EnqueueJob(job)
}

func vmssGuardName(resourceGroupName, vmssGuardName string, suffix string) string {
	return resourceGroupName + "/" + vmssGuardName + "/" + suffix
}
