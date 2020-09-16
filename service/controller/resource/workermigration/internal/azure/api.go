package azure

import (
	"context"

	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"

	"github.com/giantswarm/azure-operator/v4/client"
)

type api struct {
	clientFactory *client.Factory
	credentials   *providerv1alpha1.CredentialSecret
}

func GetAPI(f *client.Factory, credentials *providerv1alpha1.CredentialSecret) API {
	return &api{
		clientFactory: f,
		credentials:   credentials,
	}
}

func (a *api) GetVMSS(ctx context.Context, resourceGroupName, vmssName string) (VMSS, error) {
	return nil, nil
}

func (a *api) DeleteVMSS(ctx context.Context, resourceGroupName, vmssName string) error {
	return nil
}
