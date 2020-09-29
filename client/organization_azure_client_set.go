package client

import (
	"context"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v4/pkg/credential"
)

type OrganizationAzureClientSet struct {
	ctrlClient ctrlclient.Client
	logger     micrologger.Logger
	provider   credential.Provider
}

type OrganizationAzureClientSetConfig struct {
	CtrlClient ctrlclient.Client
	Logger     micrologger.Logger
	Provider   credential.Provider
}

func NewOrganizationAzureClientSet(c OrganizationAzureClientSetConfig) *OrganizationAzureClientSet {
	return &OrganizationAzureClientSet{
		ctrlClient: c.CtrlClient,
		logger:     c.Logger,
		provider:   c.Provider,
	}
}

func (f *OrganizationAzureClientSet) Get(ctx context.Context, objectMeta *v1.ObjectMeta) (*AzureClientSet, error) {
	clientCredentials, subscriptionID, partnerID, err := f.provider.GetOrganizationAzureCredentials(ctx, objectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	return NewAzureClientSet(clientCredentials, subscriptionID, partnerID)
}
