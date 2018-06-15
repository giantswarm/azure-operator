package vpngateway

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2017-09-01/network"
	"github.com/giantswarm/microerror"
	// "github.com/giantswarm/azure-operator/client"
	// "github.com/giantswarm/azure-operator/service/controller/v2/key"

	servicecontext "github.com/giantswarm/azure-operator/service/controller/v2/context"
)

// getVirtualNetworksClient return an azure client to interact with
// VirtualNetworks resources.
func (r *Resource) getVirtualNetworkGatewaysClient(ctx context.Context) (*network.VirtualNetworksClient, error) {
	sc, err := servicecontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return sc.AzureClientSet.VirtualNetworkClient, nil
}
