package azureclusterconditions

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-11-01/network"
	"github.com/giantswarm/microerror"
	capz "sigs.k8s.io/cluster-api-provider-azure/api/v1beta1"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
	capiconditions "sigs.k8s.io/cluster-api/util/conditions"

	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

const (
	PeeringNotFound     = "PeeringNotFound"
	PeeringStateUnknown = "PeeringStateUnknown"
	PeeringStateLabel   = "PeeringState"

	// TODO move constant to apiextensions.
	VNetPeeringReadyCondition = "VNetPeeringReady"
)

func (r *Resource) ensureVNetPeeringReadyCondition(ctx context.Context, azureCluster *capz.AzureCluster) error {
	r.logger.Debugf(ctx, "ensuring condition %s", VNetPeeringReadyCondition)
	var err error

	// Get Azure Deployments client
	vnetPeeringsClient, err := r.azureClientsFactory.GetVnetPeeringsClient(ctx, azureCluster.ObjectMeta)
	if err != nil {
		return microerror.Mask(err)
	}

	// Get VNet peering
	peering, err := vnetPeeringsClient.Get(ctx, key.ClusterName(azureCluster), azureCluster.Spec.NetworkSpec.Vnet.Name, r.installationName)
	if IsNotFound(err) {
		// VNet peering has not been found, which means that we still
		// didn't start deploying it.
		r.setVnetPeeringNotFound(ctx, azureCluster)
		return nil
	} else if err != nil {
		r.setProvisioningStateUnknown(ctx, azureCluster)
		return nil
	}

	// Check the peering state.

	switch peering.PeeringState {
	case network.VirtualNetworkPeeringStateConnected:
		// All good, VNet peering is connected! :)
		capiconditions.MarkTrue(azureCluster, VNetPeeringReadyCondition)
	default:
		// VNet peering is still initializing.
		r.setProvisioningStateWarning(ctx, azureCluster, string(peering.PeeringState))
	}

	r.logger.Debugf(ctx, "finished ensuring condition %s", VNetPeeringReadyCondition)

	return nil
}

func (r *Resource) setProvisioningStateWarning(ctx context.Context, azureCluster *capz.AzureCluster, currentProvisioningState string) {
	message := "VNet peering %s is not connected yet. Current PeeringState is %s, " +
		"check back in few minutes, see Azure portal for more details"
	messageArgs := []interface{}{r.installationName, currentProvisioningState}
	reason := PeeringStateLabel + currentProvisioningState

	capiconditions.MarkFalse(
		azureCluster,
		VNetPeeringReadyCondition,
		reason,
		capi.ConditionSeverityWarning,
		message,
		messageArgs...)

	r.logger.Debugf(ctx, message, messageArgs...)
}

func (r *Resource) setProvisioningStateUnknown(ctx context.Context, azureCluster *capz.AzureCluster) {
	message := "VNet peering %s PeeringState is still unknown, check back in few minutes"
	messageArgs := r.installationName
	capiconditions.MarkFalse(
		azureCluster,
		VNetPeeringReadyCondition,
		PeeringStateUnknown,
		capi.ConditionSeverityWarning,
		message,
		messageArgs)

	r.logger.Debugf(ctx, message, messageArgs)
}

func (r *Resource) setVnetPeeringNotFound(ctx context.Context, azureCluster *capz.AzureCluster) {
	message := "VNet peering %s is not found, check back in few minutes"
	messageArgs := r.installationName
	capiconditions.MarkFalse(
		azureCluster,
		VNetPeeringReadyCondition,
		PeeringNotFound,
		capi.ConditionSeverityWarning,
		message,
		messageArgs)

	r.logger.Debugf(ctx, message, messageArgs)
}
