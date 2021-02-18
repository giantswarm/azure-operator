package deployment

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-11-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	providerv1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v4/pkg/controller/context/reconciliationcanceledcontext"

	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

const (
	storageServiceEndpoint = "Microsoft.Storage"
)

func (r *Resource) ensureServiceEndpoints(ctx context.Context, cr providerv1alpha1.AzureConfig) error {
	subnetsClient, err := r.wcAzureClientFactory.GetSubnetsClient(ctx, key.ClusterID(&cr))
	if err != nil {
		return microerror.Mask(err)
	}

	subnet, err := subnetsClient.Get(ctx, key.ResourceGroupName(cr), key.VnetName(cr), key.MasterSubnetName(cr), "")
	if err != nil {
		return microerror.Mask(err)
	}

	if subnet.ServiceEndpoints != nil {
		for _, existing := range *subnet.ServiceEndpoints {
			if *existing.Service == storageServiceEndpoint {
				return nil
			}
		}
	} else {
		subnet.ServiceEndpoints = &[]network.ServiceEndpointPropertiesFormat{}
	}

	r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("service '%s' for subnet %s was not enabled. Adding it.", storageServiceEndpoint, *subnet.Name))

	*subnet.ServiceEndpoints = append(*subnet.ServiceEndpoints, network.ServiceEndpointPropertiesFormat{
		Service: to.StringPtr(storageServiceEndpoint),
	})

	_, err = subnetsClient.CreateOrUpdate(ctx, key.ResourceGroupName(cr), key.VnetName(cr), key.MasterSubnetName(cr), subnet)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("service '%s' was added to subnet %s", storageServiceEndpoint, *subnet.Name))

	reconciliationcanceledcontext.SetCanceled(ctx)
	r.logger.Debugf(ctx, "canceling reconciliation")
	return nil
}
