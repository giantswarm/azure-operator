package vpn

import (
	"context"

	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v5/service/controller/key"
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

	r.logger.Debugf(ctx, "ensuring vpn gateway")

	// Wait for virtual network subnet.
	{
		vnetName := key.VnetName(cr)
		vnet, err := vnetClient.Get(ctx, key.ClusterID(&cr), vnetName, "")
		if err != nil {
			r.logger.Debugf(ctx, "virtual network %#q not ready", vnetName)
			r.logger.Debugf(ctx, "canceling resource")
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
			r.logger.Debugf(ctx, "subnet %#q not ready", subnetName)
			r.logger.Debugf(ctx, "canceling resource")
			return nil
		}

	}

	// Prepare VPN Gateway deployment
	var deployment azureresource.Deployment
	{
		d, err := deploymentsClient.Get(ctx, key.ClusterID(&cr), vpnDeploymentName)
		if IsNotFound(err) {
			// fallthrough
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			s := *d.Properties.ProvisioningState

			r.logger.Debugf(ctx, "vpn gateway deployment is in state '%s'", s)

			if !key.IsSucceededProvisioningState(s) {
				r.debugger.LogFailedDeployment(ctx, d, err)
			}
			if !key.IsFinalProvisioningState(s) {
				r.logger.Debugf(ctx, "canceling resource")
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

	r.logger.Debugf(ctx, "ensured vpn gateway")

	return nil
}
