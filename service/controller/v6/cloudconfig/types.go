package cloudconfig

import (
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/certs"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/controller/setting"
	"github.com/giantswarm/azure-operator/service/controller/v6/key"
)

type calicoAzureFileParams struct {
	Cluster    providerv1alpha1.Cluster
	CalicoCIDR string
}

func newCalicoAzureFileParams(obj providerv1alpha1.AzureConfig, calicoCIDR string) calicoAzureFileParams {
	return calicoAzureFileParams{
		Cluster:    obj.Spec.Cluster,
		CalicoCIDR: calicoCIDR,
	}
}

type cloudProviderConfFileVMType string

type cloudProviderConfFileParams struct {
	AADClientID                 string
	AADClientSecret             string
	Cloud                       string
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

func newCloudProviderConfFileParams(azure setting.Azure, azureConfig client.AzureClientSetConfig, obj providerv1alpha1.AzureConfig) cloudProviderConfFileParams {
	return cloudProviderConfFileParams{
		AADClientID:                 azureConfig.ClientID,
		AADClientSecret:             azureConfig.ClientSecret,
		Cloud:                       azure.Cloud,
		Location:                    azure.Location,
		PrimaryScaleSetName:         key.WorkerVMSSName(obj),
		ResourceGroup:               key.ResourceGroupName(obj),
		RouteTableName:              key.RouteTableName(obj),
		SecurityGroupName:           key.WorkerSecurityGroupName(obj),
		SubnetName:                  key.WorkerSubnetName(obj),
		SubscriptionID:              azureConfig.SubscriptionID,
		TenantID:                    azureConfig.TenantID,
		VnetName:                    key.VnetName(obj),
		UseManagedIdentityExtension: azure.MSI.Enabled,
	}
}

type certificateDecrypterUnitParams struct {
	CertsPaths []string
}

type diskParams struct {
	LUNID string
}

type ingressLBFileParams struct {
	ClusterDNSDomain string
}

func newCertificateDecrypterUnitParams(certFiles certs.Files) certificateDecrypterUnitParams {
	var certsPaths []string

	for _, file := range certFiles {
		certsPaths = append(certsPaths, file.AbsolutePath)
	}

	return certificateDecrypterUnitParams{
		CertsPaths: certsPaths,
	}
}

func newIngressLBFileParams(obj providerv1alpha1.AzureConfig) ingressLBFileParams {
	return ingressLBFileParams{
		ClusterDNSDomain: key.ClusterDNSDomain(obj),
	}
}
