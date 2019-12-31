package vpnconnection

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-06-01/network"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/resource/crud"

	"github.com/giantswarm/azure-operator/service/controller/v9/key"
)

// NewUpdatePatch provide a crud.Patch holding the needed connections update.
func (r *Resource) NewUpdatePatch(ctx context.Context, azureConfig, current, desired interface{}) (*crud.Patch, error) {
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

func (r *Resource) newUpdatePatch(ctx context.Context, azureConfig providerv1alpha1.AzureConfig, current, desired connections) (*crud.Patch, error) {
	patch := crud.NewPatch()

	change := r.newUpdateChange(ctx, azureConfig, current, desired)

	patch.SetUpdateChange(change)

	return patch, nil
}

func (r *Resource) newUpdateChange(ctx context.Context, azureConfig providerv1alpha1.AzureConfig, current, desired connections) connections {
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
	r.logger.LogCtx(ctx, "level", "debug", "message", "ensuring vpn gateway connections are created")

	if change.isEmpty() {
		r.logger.LogCtx(ctx, "level", "debug", "message", "ensured vpn gateway connections are created")
		return nil
	}

	{
		hostGatewayConnectionClient, err := r.getHostVirtualNetworkGatewayConnectionsClient(ctx)
		if err != nil {
			return microerror.Mask(err)
		}

		resourceGroup := r.azure.HostCluster.ResourceGroup
		connectionName := *change.Host.Name
		connection := change.Host
		res, err := hostGatewayConnectionClient.CreateOrUpdate(ctx, resourceGroup, connectionName, connection)
		if err != nil {
			return microerror.Mask(err)
		}
		_, err = hostGatewayConnectionClient.CreateOrUpdateResponder(res.Response())
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
