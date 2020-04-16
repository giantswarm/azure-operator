package vmsscheck

import "context"

type InstanceWatchdog interface {
	GuardVMSS(ctx context.Context, resourceGroupName, vmssName string)
}
