package vpngateway

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2017-09-01/network"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller"

	"github.com/giantswarm/azure-operator/service/controller/v2/key"
)

// NewUpdatePatch provide a controller.Patch holding the needed connections update.
func (r *Resource) NewUpdatePatch(ctx context.Context, azureConfig, current, desired interface{}) (*controller.Patch, error) {
	a, err := key.ToCustomObject(azureConfig)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	c, err := toVPNGatewayConnections(current)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	d, err := toVPNGatewayConnections(desired)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	patch, err := r.newUpdatePatch(ctx, a, c, d)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return patch, nil
}

func (r *Resource) newUpdatePatch(ctx context.Context, azureConfig providerv1alpha1.AzureConfig, current, desired connections) (*controller.Patch, error) {
	patch := controller.NewPatch()

	change := r.newUpdateChange(ctx, azureConfig, current, desired)

	patch.SetUpdateChange(change)

	return patch, nil
}

func (r *Resource) newUpdateChange(ctx context.Context, azureConfig providerv1alpha1.AzureConfig, current, desired connections) connections {
	var change connections

	if needUpdate(current.Host, desired.Host) {
		change = desired
	}
	if needUpdate(current.Guest, desired.Guest) {
		change = desired
	}

	return change
}

// needUpdate determine if current needs to be updated in order to comply with
// desired. Following properties are examined:
//
//     Name
//     VirtualNetworkGatewayConnectionPropertiesFormat.VirtualNetworkGateway1.ID
//     VirtualNetworkGatewayConnectionPropertiesFormat.VirtualNetworkGateway2.ID
//     VirtualNetworkGatewayConnectionPropertiesFormat.ConnectionType
//     VirtualNetworkGatewayConnectionPropertiesFormat.ConnectionStatus
//
func needUpdate(current, desired network.VirtualNetworkGatewayConnection) bool {
	if desired.Name == nil ||
		desired.VirtualNetworkGatewayConnectionPropertiesFormat == nil ||
		desired.VirtualNetworkGatewayConnectionPropertiesFormat.VirtualNetworkGateway1 == nil ||
		desired.VirtualNetworkGatewayConnectionPropertiesFormat.VirtualNetworkGateway1.ID == nil ||
		desired.VirtualNetworkGatewayConnectionPropertiesFormat.VirtualNetworkGateway2 == nil ||
		desired.VirtualNetworkGatewayConnectionPropertiesFormat.VirtualNetworkGateway2.ID == nil {
		return false
	}

	if current.Name == nil ||
		*current.Name != *desired.Name {
		return true
	}

	if current.VirtualNetworkGatewayConnectionPropertiesFormat == nil {
		return true
	}

	if current.VirtualNetworkGatewayConnectionPropertiesFormat.ConnectionType !=
		desired.VirtualNetworkGatewayConnectionPropertiesFormat.ConnectionType {
		return true
	}

	if current.VirtualNetworkGatewayConnectionPropertiesFormat.VirtualNetworkGateway1 == nil ||
		current.VirtualNetworkGatewayConnectionPropertiesFormat.VirtualNetworkGateway1.ID == nil ||
		*current.VirtualNetworkGatewayConnectionPropertiesFormat.VirtualNetworkGateway1.ID !=
			*desired.VirtualNetworkGatewayConnectionPropertiesFormat.VirtualNetworkGateway1.ID {
		return true
	}

	if current.VirtualNetworkGatewayConnectionPropertiesFormat.VirtualNetworkGateway2 == nil ||
		current.VirtualNetworkGatewayConnectionPropertiesFormat.VirtualNetworkGateway2.ID == nil ||
		*current.VirtualNetworkGatewayConnectionPropertiesFormat.VirtualNetworkGateway2.ID !=
			*desired.VirtualNetworkGatewayConnectionPropertiesFormat.VirtualNetworkGateway2.ID {
		return true
	}

	if current.VirtualNetworkGatewayConnectionPropertiesFormat.ConnectionStatus == network.VirtualNetworkGatewayConnectionStatusNotConnected {
		return true
	}

	return false
}

// ApplyUpdateChange perform the host cluster vpn gateway connections update against azure.
func (r *Resource) ApplyUpdateChange(ctx context.Context, azureConfig, change interface{}) error {
	a, err := key.ToCustomObject(azureConfig)
	if err != nil {
		return microerror.Mask(err)
	}

	c, err := toVPNGatewayConnections(change)
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.applyUpdateChange(ctx, a, c)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *Resource) applyUpdateChange(ctx context.Context, azureConfig providerv1alpha1.AzureConfig, change connections) error {
	r.logger.LogCtx(ctx, "level", "debug", "message", "ensure vpn gateway connections")

	if change.isEmpty() {
		r.logger.LogCtx(ctx, "level", "debug", "message", "ensure vpn gateway connections: already ensured")
		return nil
	}

	hostGatewayConnectionClient, err := r.getHostVirtualNetworkGatewayConnectionsClient(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	guestGatewayConnectionClient, err := r.getGuestVirtualNetworkGatewayConnectionsClient(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	resourceGroup := r.azure.HostCluster.ResourceGroup
	connectionName := *change.Host.Name
	connection := change.Host
	_, err = hostGatewayConnectionClient.CreateOrUpdate(ctx, resourceGroup, connectionName, connection)
	if err != nil {
		return microerror.Mask(err)
	}

	resourceGroup = key.ResourceGroupName(azureConfig)
	connectionName = *change.Guest.Name
	connection = change.Guest
	_, err = guestGatewayConnectionClient.CreateOrUpdate(ctx, resourceGroup, connectionName, connection)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "ensure vpn gateway connections: created")
	return nil
}
