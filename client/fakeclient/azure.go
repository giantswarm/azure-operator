package fakeclient

import (
	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/micrologger/microloggertest"
)

// TODO(PK) remove it as soon as we sort AzureClient and calico CC extention.
func NewAzureConfig() client.AzureConfig {
	return client.AzureConfig{
		Logger:         microloggertest.New(),
		ClientID:       "fakeClientID",
		ClientSecret:   "fakeClientSecret",
		SubscriptionID: "fakeSubscriptionID",
		TenantID:       "fakeTenantID",
	}
}
