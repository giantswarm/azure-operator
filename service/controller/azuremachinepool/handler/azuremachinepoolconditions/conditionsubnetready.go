package azuremachinepoolconditions

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-11-01/network"
	azureconditions "github.com/giantswarm/apiextensions/v6/pkg/conditions/azure"
	"github.com/giantswarm/microerror"
	capzexp "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1beta1"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
	capiconditions "sigs.k8s.io/cluster-api/util/conditions"

	"github.com/giantswarm/azure-operator/v5/pkg/helpers"
	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

const (
	SubnetNotFoundReason          = "SubnetNotFound"
	SubnetProvisioningStatePrefix = "SubnetProvisioningState"
)

func (r *Resource) ensureSubnetReadyCondition(ctx context.Context, azureMachinePool *capzexp.AzureMachinePool) error {
	r.logger.Debugf(ctx, "ensuring condition %s", azureconditions.SubnetReadyCondition)

	deploymentsClient, err := r.azureClientsFactory.GetDeploymentsClient(ctx, azureMachinePool.ObjectMeta)
	if err != nil {
		return microerror.Mask(err)
	}

	// Now let's first check ARM deployment state
	subnetDeploymentName := key.SubnetDeploymentName(azureMachinePool.Name)
	isSubnetDeploymentSuccessful, err := r.deploymentChecker.CheckIfDeploymentIsSuccessful(ctx, deploymentsClient, azureMachinePool, subnetDeploymentName, azureconditions.SubnetReadyCondition)
	if err != nil {
		return microerror.Mask(err)
	} else if !isSubnetDeploymentSuccessful {
		// Function CheckIfDeploymentIsSuccessful that is called above, if it
		// sees that the deployment is not succeeded, for whatever reason, it
		// will also set appropriate condition value, so our job here is done.
		return nil
	}

	// Deployment is successful, we proceed with checking the actual Azure
	// subnet.
	subnetsClient, err := r.azureClientsFactory.GetSubnetsClient(ctx, azureMachinePool.ObjectMeta)
	if err != nil {
		return microerror.Mask(err)
	}

	azureCluster, err := helpers.GetAzureClusterFromMetadata(ctx, r.ctrlClient, azureMachinePool.ObjectMeta)
	if err != nil {
		return microerror.Mask(err)
	}

	subnetName := azureMachinePool.Name
	subnet, err := subnetsClient.Get(ctx, azureCluster.Name, azureCluster.Spec.NetworkSpec.Vnet.Name, subnetName, "")
	if IsNotFound(err) {
		r.setSubnetNotFound(ctx, azureMachinePool, subnetName, azureconditions.SubnetReadyCondition)
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	// Note: Here we check if the subnet exists and that its provisioning state
	// is succeeded. It would be good to also check network security group,
	// routing table and service endpoints.
	if subnet.ProvisioningState == network.Succeeded {
		capiconditions.MarkTrue(azureMachinePool, azureconditions.SubnetReadyCondition)
	} else {
		r.setSubnetProvisioningStateNotSuccessful(ctx, azureMachinePool, subnetName, subnet.ProvisioningState, azureconditions.SubnetReadyCondition)
	}

	r.logConditionStatus(ctx, azureMachinePool, azureconditions.SubnetReadyCondition)
	r.logger.Debugf(ctx, "ensured condition %s", azureconditions.SubnetReadyCondition)
	return nil
}

func (r *Resource) setSubnetNotFound(ctx context.Context, cr capiconditions.Setter, subnetName string, condition capi.ConditionType) {
	message := "Subnet %s is not found"
	messageArgs := subnetName
	capiconditions.MarkFalse(
		cr,
		condition,
		SubnetNotFoundReason,
		capi.ConditionSeverityError,
		message,
		messageArgs)

	r.logger.Debugf(ctx, message, messageArgs)
}

func (r *Resource) setSubnetProvisioningStateNotSuccessful(ctx context.Context, cr capiconditions.Setter, subnetName string, provisioningState network.ProvisioningState, condition capi.ConditionType) {
	message := "Subnet %s provisioning state is %s"
	messageArgs := []interface{}{subnetName, provisioningState}
	reason := SubnetProvisioningStatePrefix + string(provisioningState)

	capiconditions.MarkFalse(
		cr,
		condition,
		reason,
		capi.ConditionSeverityWarning,
		message,
		messageArgs...)

	r.logger.Debugf(ctx, message, messageArgs...)
}
