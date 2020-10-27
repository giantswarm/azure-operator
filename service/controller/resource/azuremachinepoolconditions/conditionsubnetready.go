package azuremachinepoolconditions

import (
	"context"
	"fmt"

	azureconditions "github.com/giantswarm/apiextensions/v3/pkg/conditions/azure"
	"github.com/giantswarm/microerror"
	capzexp "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
	capiconditions "sigs.k8s.io/cluster-api/util/conditions"

	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

const (
	subnetDeploymentPrefix = "subnet"
)

func (r *Resource) ensureSubnetReadyCondition(ctx context.Context, azureMachinePool *capzexp.AzureMachinePool) error {
	r.logDebug(ctx, "ensuring condition %s", azureconditions.SubnetReadyCondition)

	r.logConditionStatus(ctx, azureMachinePool, azureconditions.SubnetReadyCondition)
	r.logDebug(ctx, "ensured condition %s", azureconditions.SubnetReadyCondition)
	return nil
}

func (r *Resource) checkSubnetDeployment(ctx context.Context, azureMachinePool *capzexp.AzureMachinePool) error {
	// Get Azure Deployments client
	deploymentsClient, err := r.azureClientsFactory.GetDeploymentsClient(ctx, azureMachinePool.ObjectMeta)
	if err != nil {
		return microerror.Mask(err)
	}

	subnetDeploymentName := getSubnetDeploymentName(azureMachinePool.Name)
	deployment, err := deploymentsClient.Get(ctx, key.ClusterName(azureMachinePool), subnetDeploymentName)
	if IsNotFound(err) {
		// Subnet deployment has not been found, which means that we still
		// didn't start deploying it.
		r.setDeploymentNotFound(ctx, azureMachinePool, subnetDeploymentName, azureconditions.SubnetReadyCondition)
		return nil
	} else if err != nil {
		// Error while getting Subnet deployment, let's check if
		// deployment provisioning state is set.
		if !isProvisioningStateSet(&deployment) {
			return microerror.Mask(err)
		}

		currentProvisioningState := *deployment.Properties.ProvisioningState
		r.setProvisioningStateWarning(ctx, azureMachinePool, currentProvisioningState, subnetDeploymentName, azureconditions.SubnetReadyCondition)
		return nil
	}

	// We got the Subnet deployment without errors, but for some reason the provisioning state is
	// not set.
	if !isProvisioningStateSet(&deployment) {
		r.setProvisioningStateUnknown(ctx, azureMachinePool, subnetDeploymentName, azureconditions.SubnetReadyCondition)
		return nil
	}

	// Now let's finally check what's the current subnet deployment
	// provisioning state.
	currentProvisioningState := *deployment.Properties.ProvisioningState

	switch currentProvisioningState {
	case DeploymentProvisioningStateSucceeded:
		// All good, Subnet deployment has been completed successfully! :)
		capiconditions.MarkTrue(azureMachinePool, azureconditions.SubnetReadyCondition)
	case DeploymentProvisioningStateFailed:
		// Subnet deployment has failed.
		r.setProvisioningStateWarningFailed(ctx, azureMachinePool, subnetDeploymentName, azureconditions.SubnetReadyCondition)
	default:
		// Subnet deployment is probably still running.
		r.setProvisioningStateWarning(ctx, azureMachinePool, currentProvisioningState, subnetDeploymentName, azureconditions.SubnetReadyCondition)
	}

	r.logDebug(ctx, "finished ensuring condition %s", azureconditions.SubnetReadyCondition)

	return nil
}

func getSubnetDeploymentName(subnetName string) string {
	return fmt.Sprintf("%s-%s", subnetDeploymentPrefix, subnetName)
}
