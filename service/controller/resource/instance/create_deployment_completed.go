package instance

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	"sigs.k8s.io/cluster-api/util"

	"github.com/giantswarm/azure-operator/v4/pkg/checksum"
	"github.com/giantswarm/azure-operator/v4/service/controller/blobclient"
	"github.com/giantswarm/azure-operator/v4/service/controller/internal/state"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

func (r *Resource) deploymentCompletedTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	azureMachinePool, err := key.ToAzureMachinePool(obj)
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	machinePool, err := r.getOwnerMachinePool(ctx, azureMachinePool.ObjectMeta)
	if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}

	cluster, err := util.GetClusterFromMetadata(ctx, r.CtrlClient, azureMachinePool.ObjectMeta)
	if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}

	azureCluster, err := r.getAzureClusterFromCluster(ctx, cluster)
	if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}

	release, err := r.getReleaseFromMetadata(ctx, azureMachinePool.ObjectMeta)
	if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}

	credentialSecret, err := r.getCredentialSecret(ctx, *cluster)
	if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}

	deploymentsClient, err := r.ClientFactory.GetDeploymentsClient(credentialSecret.Namespace, credentialSecret.Name)
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	storageAccountsClient, err := r.ClientFactory.GetStorageAccountsClient(credentialSecret.Namespace, credentialSecret.Name)
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	deployment, err := deploymentsClient.Get(ctx, key.ClusterID(&azureMachinePool), key.WorkersVmssDeploymentName)
	if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}

	provisioningState := *deployment.Properties.ProvisioningState
	r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deployment is in state '%s'", provisioningState))

	if key.IsSucceededProvisioningState(provisioningState) {
		computedDeployment, err := r.newDeployment(ctx, storageAccountsClient, release, *machinePool, azureMachinePool, azureCluster)
		if blobclient.IsBlobNotFound(err) {
			r.Logger.LogCtx(ctx, "level", "debug", "message", "ignition blob not found")
			return currentState, nil
		} else if err != nil {
			return DeploymentUninitialized, microerror.Mask(err)
		} else {
			desiredDeploymentTemplateChk, err := checksum.GetDeploymentTemplateChecksum(computedDeployment)
			if err != nil {
				return DeploymentUninitialized, microerror.Mask(err)
			}

			desiredDeploymentParametersChk, err := checksum.GetDeploymentParametersChecksum(computedDeployment)
			if err != nil {
				return DeploymentUninitialized, microerror.Mask(err)
			}

			currentDeploymentTemplateChk, err := r.GetResourceStatus(azureMachinePool, DeploymentTemplateChecksum)
			if err != nil {
				return DeploymentUninitialized, microerror.Mask(err)
			}

			currentDeploymentParametersChk, err := r.GetResourceStatus(azureMachinePool, DeploymentParametersChecksum)
			if err != nil {
				return DeploymentUninitialized, microerror.Mask(err)
			}

			if currentDeploymentTemplateChk != desiredDeploymentTemplateChk || currentDeploymentParametersChk != desiredDeploymentParametersChk {
				r.Logger.LogCtx(ctx, "level", "debug", "message", "template or parameters changed")
				// As current and desired state differs, start process from the beginning.
				return DeploymentUninitialized, nil
			}

			r.Logger.LogCtx(ctx, "level", "debug", "message", "template and parameters unchanged")

			return currentState, nil
		}
	} else if key.IsFinalProvisioningState(provisioningState) {
		// Deployment has failed. Restart from beginning.
		return DeploymentUninitialized, nil
	}

	r.Logger.LogCtx(ctx, "level", "warning", "message", "instances reconciliation process reached unexpected state")

	// Normally the process should never get here. In case this happens, start
	// from the beginning.
	return DeploymentUninitialized, nil
}
