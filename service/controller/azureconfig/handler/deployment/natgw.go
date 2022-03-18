package deployment

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-11-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	providerv1alpha1 "github.com/giantswarm/apiextensions/v5/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

// This function ensures that the Master subnet has a nat gateway attached as expected.
// This is needed because we apply the Virtual Network ARM deployment only once, so upgraded clusters
// would not get the nat gateway attached to their masters without this function.
// It can be deleted once all tenant clusters will have the nat gateway enabled.
func (r *Resource) ensureNatGatewayForMasterSubnet(ctx context.Context, cr providerv1alpha1.AzureConfig) error {
	subnetsClient, err := r.clientFactory.GetSubnetsClient(ctx, cr.ObjectMeta)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("Checking if subnet %s has the nat gateway set as expected", key.MasterSubnetName(cr)))

	subnet, err := subnetsClient.Get(ctx, key.ResourceGroupName(cr), key.VnetName(cr), key.MasterSubnetName(cr), "")
	if err != nil {
		return microerror.Mask(err)
	}

	if subnet.ProvisioningState != "Succeeded" {
		r.logger.Debugf(ctx, "Subnet %s is not provisioned yet. Waiting.", key.MasterSubnetName(cr))
		return nil
	}

	if subnet.NatGateway == nil {
		r.logger.LogCtx(ctx, "level", "warning", "message", fmt.Sprintf("NAT gateway not set for subnet %s", key.MasterSubnetName(cr)))

		subnet.NatGateway = &network.SubResource{
			ID: to.StringPtr(key.MasterNatGatewayID(cr, subnetsClient.SubscriptionID)),
		}

		_, err := subnetsClient.CreateOrUpdate(ctx, key.ResourceGroupName(cr), key.VnetName(cr), key.MasterSubnetName(cr), subnet)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("NAT gateway set for subnet %s", key.MasterSubnetName(cr)))
		return nil
	}

	r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("NAT gateway was already set for subnet %s", key.MasterSubnetName(cr)))

	return nil
}
