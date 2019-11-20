package client

import (
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/giantswarm/microerror"
)

type AzureClientSetConfig struct {
	// ClientID is the ID of the Active Directory Service Principal.
	ClientID string
	// ClientSecret is the secret of the Active Directory Service Principal.
	ClientSecret string
	// EnvironmentName is the cloud environment identifier on Azure. Values can be
	// used as listed in the link below.
	//
	//     https://github.com/Azure/go-autorest/blob/ec5f4903f77ed9927ac95b19ab8e44ada64c1356/autorest/azure/environments.go#L13
	//
	EnvironmentName string
	// SubscriptionID is the ID of the Azure subscription.
	SubscriptionID string
	// TenantID is the ID of the Active Directory tenant.
	TenantID string
	// PartnerID is the ID used for the Azure Partner Program.
	PartnerID string
}

func (c AzureClientSetConfig) Validate() error {
	if c.ClientID == "" {
		return microerror.Maskf(invalidConfigError, "%T.ClientID must not be empty", c)
	}
	if c.ClientSecret == "" {
		return microerror.Maskf(invalidConfigError, "%T.ClientSecret must not be empty", c)
	}
	if c.SubscriptionID == "" {
		return microerror.Maskf(invalidConfigError, "%T.SubscriptionID must not be empty", c)
	}
	if c.TenantID == "" {
		return microerror.Maskf(invalidConfigError, "%T.TenantID must not be empty", c)
	}

	return nil
}

// clientConfig contains all essential information to create an Azure client.
type clientConfig struct {
	subscriptionID          string
	partnerIdUserAgent      string
	resourceManagerEndpoint string
	servicePrincipalToken   *adal.ServicePrincipalToken
}
