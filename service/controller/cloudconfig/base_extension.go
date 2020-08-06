package cloudconfig

import (
	"github.com/Azure/go-autorest/autorest/azure/auth"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/certs/v2/pkg/certs"

	"github.com/giantswarm/azure-operator/v4/service/controller/encrypter"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
	"github.com/giantswarm/azure-operator/v4/service/controller/setting"
)

type baseExtension struct {
	azure                        setting.Azure
	azureClientCredentialsConfig auth.ClientCredentialsConfig
	calicoCIDR                   string
	certFiles                    []certs.File
	customObject                 providerv1alpha1.AzureConfig
	encrypter                    encrypter.Interface
	subscriptionID               string
	vnetCIDR                     string
}

func (e *baseExtension) templateData(certFiles []certs.File) templateData {
	var certsPaths []string

	for _, file := range certFiles {
		certsPaths = append(certsPaths, file.AbsolutePath)
	}

	return templateData{
		azureCNIFileParams{
			VnetCIDR: e.vnetCIDR,
		},
		calicoAzureFileParams{
			Cluster:    e.customObject.Spec.Cluster,
			CalicoCIDR: e.calicoCIDR,
		},
		cloudProviderConfFileParams{
			AADClientID:                 e.azureClientCredentialsConfig.ClientID,
			AADClientSecret:             e.azureClientCredentialsConfig.ClientSecret,
			EnvironmentName:             e.azure.EnvironmentName,
			Location:                    e.azure.Location,
			PrimaryScaleSetName:         key.WorkerVMSSName(e.customObject),
			ResourceGroup:               key.ResourceGroupName(e.customObject),
			RouteTableName:              key.RouteTableName(e.customObject),
			SecurityGroupName:           key.WorkerSecurityGroupName(e.customObject),
			SubnetName:                  key.WorkerSubnetName(e.customObject),
			SubscriptionID:              e.subscriptionID,
			TenantID:                    e.azureClientCredentialsConfig.TenantID,
			VnetName:                    key.VnetName(e.customObject),
			UseManagedIdentityExtension: e.azure.MSI.Enabled,
		},
		certificateDecrypterUnitParams{
			CertsPaths: certsPaths,
		},
		ingressLBFileParams{
			ClusterDNSDomain: key.ClusterDNSDomain(e.customObject),
		},
	}
}
