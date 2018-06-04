package fakeclient

import (
	"github.com/giantswarm/azure-operator/client"
)

// TODO(PK) remove it as soon as we sort AzureClient and calico CC extention.
func NewAzureConfig() client.AzureClientSetConfig {
	return client.AzureClientSetConfig{
		ClientID:       "fakeClientID",
		ClientSecret:   "fakeClientSecret",
		SubscriptionID: "fakeSubscriptionID",
		TenantID:       "fakeTenantID",
	}
}
