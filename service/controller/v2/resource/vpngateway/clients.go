package vpngateway

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2017-10-01/network"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller/context/resourcecanceledcontext"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/controller/v2/key"
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
