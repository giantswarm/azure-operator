package cloudconfig

import (
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
)

type templateData struct {
	azureCNIFileParams
	calicoAzureFileParams
	cloudProviderConfFileParams
	certificateDecrypterUnitParams
	ingressLBFileParams
}

type azureCNIFileParams struct {
	VnetCIDR string
}

type calicoAzureFileParams struct {
	Cluster    providerv1alpha1.Cluster
	CalicoCIDR string
}

type cloudProviderConfFileVMType string

type cloudProviderConfFileParams struct {
	AADClientID                 string
	AADClientSecret             string
	EnvironmentName             string
	Location                    string
	PrimaryScaleSetName         string
	ResourceGroup               string
	RouteTableName              string
	SecurityGroupName           string
	SubnetName                  string
	SubscriptionID              string
	TenantID                    string
	VnetName                    string
	UseManagedIdentityExtension bool
}

type certificateDecrypterUnitParams struct {
	CertsPaths []string
}

type ingressLBFileParams struct {
	ClusterDNSDomain string
	PublicIPName     string
}
