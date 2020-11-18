package vpnconnection

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-11-01/network"
	providerv1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v4/pkg/resource/crud"

	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

// NewUpdatePatch provide a crud.Patch holding the needed connections update.
func (r *Resource) NewUpdatePatch(ctx context.Context, azureConfig, current, desired interface{}) (*crud.Patch, error) {
	c, err := toVPNGatewayConnections(current)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	d, err := toVPNGatewayConnections(desired)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return r.newUpdatePatch(c, d), nil
}

func (r *Resource) newUpdatePatch(current, desired connections) *crud.Patch {
	patch := crud.NewPatch()
	change := r.newUpdateChange(current, desired)
	patch.SetUpdateChange(change)

	return patch
}

func (r *Resource) newUpdateChange(current, desired connections) connections {
	var change connections

	if needsUpdate(current.Host, desired.Host) {
		change = desired
	}
	if needsUpdate(current.Guest, desired.Guest) {
		change = desired
	}

	return change
}

// needsUpdate determine if current needs to be updated in order to comply with
// desired. Following properties are examined:
//
//     Name
//     VirtualNetworkGatewayConnectionPropertiesFormat.VirtualNetworkGateway1.ID
//     VirtualNetworkGatewayConnectionPropertiesFormat.VirtualNetworkGateway2.ID
//     VirtualNetworkGatewayConnectionPropertiesFormat.ConnectionType
//     VirtualNetworkGatewayConnectionPropertiesFormat.ConnectionStatus
//
func needsUpdate(current, desired network.VirtualNetworkGatewayConnection) bool {
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
	cr, err := key.ToCustomResource(azureConfig)
	if err != nil {
		return microerror.Mask(err)
	}

	c, err := toVPNGatewayConnections(change)
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.applyUpdateChange(ctx, cr, c)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *Resource) applyUpdateChange(ctx context.Context, azureConfig providerv1alpha1.AzureConfig, change connections) error {
	r.logger.LogCtx(ctx, "level", "debug", "message", "ensuring vpn gateway connections are created")

	if change.isEmpty() {
		r.logger.LogCtx(ctx, "level", "debug", "message", "ensured vpn gateway connections are created")
		return nil
	}

	{
		resourceGroup := r.azure.HostCluster.ResourceGroup
		connectionName := *change.Host.Name
		connection := change.Host
		res, err := r.cpVirtualNetworkGatewayConnectionsClient.CreateOrUpdate(ctx, resourceGroup, connectionName, connection)
		if err != nil {
			return microerror.Mask(err)
		}
		_, err = r.cpVirtualNetworkGatewayConnectionsClient.CreateOrUpdateResponder(res.Response())
		if err != nil {
			return microerror.Mask(err)
		}
	}

	{
		guestGatewayConnectionClient, err := r.getGuestVirtualNetworkGatewayConnectionsClient(ctx)
		if err != nil {
			return microerror.Mask(err)
		}

		resourceGroup := key.ResourceGroupName(azureConfig)
		connectionName := *change.Guest.Name
		connection := change.Guest
		res, err := guestGatewayConnectionClient.CreateOrUpdate(ctx, resourceGroup, connectionName, connection)
		if err != nil {
			return microerror.Mask(err)
		}
		_, err = guestGatewayConnectionClient.CreateOrUpdateResponder(res.Response())
		if err != nil {
			return microerror.Mask(err)
		}
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "ensured vpn gateway connections are created")

	return nil
}
