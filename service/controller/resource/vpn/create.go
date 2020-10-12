package vpn

import (
	"context"
	"fmt"

	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

const (
	vpnDeploymentName = "vpn-template"
)

// EnsureCreated ensures the resource is created.
func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	cr, err := key.ToCustomResource(obj)
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
		vnetName := key.VnetName(cr)
		vnet, err := vnetClient.Get(ctx, key.ClusterID(&cr), vnetName, "")
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
		d, err := deploymentsClient.Get(ctx, key.ClusterID(&cr), vpnDeploymentName)
		if IsNotFound(err) {
			conditionsUpdateError := r.UpdateVPNGatewayReadyCondition(ctx, cr, nil)
			if conditionsUpdateError != nil {
				r.logger.LogCtx(ctx, "level", "warning", "message", "error while updating conditions", "error", conditionsUpdateError.Error())
			}
			// fallthrough
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			conditionsUpdateError := r.UpdateVPNGatewayReadyCondition(ctx, cr, d.Properties.ProvisioningState)
			if conditionsUpdateError != nil {
				r.logger.LogCtx(ctx, "level", "warning", "message", "error while updating conditions", "error", conditionsUpdateError.Error())
			}

			s := *d.Properties.ProvisioningState

			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("vpn gateway deployment is in state '%s'", s))

			if !key.IsSucceededProvisioningState(s) {
				r.debugger.LogFailedDeployment(ctx, d, err)
			}
			if !key.IsFinalProvisioningState(s) {
				r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
				return nil
			}
		}

		deployment, err = r.newDeployment(cr, nil)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	// Create/Update VPN Gateway deployment
	res, err := deploymentsClient.CreateOrUpdate(ctx, key.ClusterID(&cr), vpnDeploymentName, deployment)
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
