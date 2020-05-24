package masters

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/giantswarm/microerror"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/azure-operator/v4/service/controller/controllercontext"
	"github.com/giantswarm/azure-operator/v4/service/controller/internal/state"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

func (r *Resource) restartKubeletOnWorkersTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	// Check if API server is up or wait.
	r.Logger().LogCtx(ctx, "level", "debug", "message", "Checking if API server is up")
	up, err := r.isApiServerUP(ctx)
	if err != nil {
		r.Logger().LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("API server is NOT up. Original error was %s", err.Error()))
		return currentState, nil
	}
	if !up {
		r.Logger().LogCtx(ctx, "level", "debug", "message", "API server is NOT up.")
		return currentState, nil
	}
	r.Logger().LogCtx(ctx, "level", "debug", "message", "API server is up")

	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return "", microerror.Mask(err)
	}

	groupsClient, err := r.GetGroupsClient(ctx)
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	group, err := groupsClient.Get(ctx, key.ClusterID(&cr))
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	vmssVMsClient, err := r.GetVMsClient(ctx)
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

	allMasterInstances, err := r.AllInstances(ctx, cr, key.LegacyWorkerVMSSName)
	if err != nil {
		return "", microerror.Mask(err)
	}

	for _, instance := range allMasterInstances {
		r.Logger().LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Sending restart kubelet command to %s", *instance.Name))
		runCommandFuture, err := vmssVMsClient.RunCommand(ctx, *group.Name, key.LegacyWorkerVMSSName(cr), *instance.InstanceID, runCommandInput)
		if err != nil {
			return "", microerror.Mask(err)
		}

		_, err = vmssVMsClient.RunCommandResponder(runCommandFuture.Response())
		if err != nil {
			return "", microerror.Mask(err)
		}

		r.Logger().LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Sent restart kubelet command to %s", *instance.Name))
	}

	return DeploymentCompleted, nil
}

func (r *Resource) isApiServerUP(ctx context.Context) (bool, error) {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return false, microerror.Mask(err)
	}

	if cc.Client.TenantCluster.K8s == nil {
		return false, clientNotFoundError
	}

	_, err = cc.Client.TenantCluster.K8s.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return false, microerror.Mask(err)
	}

	return true, nil
}
