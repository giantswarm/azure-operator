package masters

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/service/controller/internal/state"
	"github.com/giantswarm/azure-operator/service/controller/key"
)

func (r *Resource) restartKubeletOnWorkersTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return "", microerror.Mask(err)
	}

	groupsClient, err := r.getGroupsClient(ctx)
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	group, err := groupsClient.Get(ctx, key.ClusterID(cr))
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	vmssVMsClient, err := r.getVMsClient(ctx)
	if err != nil {
		return "", microerror.Mask(err)
	}

	commandId := "RunShellScript"
	script := []string{
		"sudo systemctl restart k8s-kubelet",
	}
	runCommandInput := compute.RunCommandInput{
		CommandID: &commandId,
		Script:    &script,
	}

	allMasterInstances, err := r.allInstances(ctx, cr, key.LegacyWorkerVMSSName)
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	for _, instance := range allMasterInstances {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Sending restart kubelet command to %s", *instance.Name))
		runCommandFuture, err := vmssVMsClient.RunCommand(ctx, *group.Name, key.LegacyWorkerVMSSName(cr), *instance.InstanceID, runCommandInput)
		if err != nil {
			return currentState, microerror.Mask(err)
		}

		_, err = vmssVMsClient.RunCommandResponder(runCommandFuture.Response())
		if err != nil {
			return currentState, microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Sent restart kubelet command to %s", *instance.Name))
	}

	return DeploymentCompleted, nil
}
