package cloudconfig

import (
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/certs"
	"github.com/giantswarm/k8scloudconfig/v_6_0_0"
	"github.com/giantswarm/randomkeys"
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
}

type IgnitionTemplateData struct {
	CustomObject providerv1alpha1.AzureConfig
	ClusterCerts certs.Cluster
	ClusterKeys  randomkeys.Cluster
	Images       v_6_0_0.Images
}
