package vpn

import (
	"context"
	"fmt"

	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/service/controller/v10patch1/key"
)

const (
	vpnDeploymentName = "vpn-template"
)

// EnsureCreated ensures the resource is created.
func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	vnetClient, err := r.getVirtualNetworkClient(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	deploymentsClient, err := r.getDeploymentsClient(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "ensuring vpn gateway")

	// Wait for virtual network subnet.
	{
		vnetName := key.VnetName(customObject)
		vnet, err := vnetClient.Get(ctx, key.ClusterID(customObject), vnetName, "")
		if err != nil {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("virtual network %#q not ready", vnetName))
			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
			return nil
		}

		found := false
		subnetName := key.VNetGatewaySubnetName()
		for _, subnet := range *vnet.Subnets {
			if *subnet.Name == subnetName {
				found = true
			}
		}
		if !found {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("subnet %#q not ready", subnetName))
			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
			return nil
		}

	}

	// Prepare VPN Gateway deployment
	var deployment azureresource.Deployment
	{
		d, err := deploymentsClient.Get(ctx, key.ClusterID(customObject), vpnDeploymentName)
		if IsNotFound(err) {
			// fallthrough
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			s := *d.Properties.ProvisioningState

			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("vpn gateway deployment is in state '%s'", s))

			if !key.IsSucceededProvisioningState(s) {
				r.debugger.LogFailedDeployment(ctx, d)
			}
			if !key.IsFinalProvisioningState(s) {
				r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
				return nil
			}
		}

		deployment = r.newDeployment(customObject, nil)
	}

	// Create/Update VPN Gateway deployment
	res, err := deploymentsClient.CreateOrUpdate(ctx, key.ClusterID(customObject), vpnDeploymentName, deployment)
	if err != nil {
		return microerror.Mask(err)
	}

	_, err = deploymentsClient.CreateOrUpdateResponder(res.Response())
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "ensured vpn gateway")

	return nil
}
