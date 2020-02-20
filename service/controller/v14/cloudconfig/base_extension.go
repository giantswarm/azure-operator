package cloudconfig

import (
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/certs"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/controller/setting"
	"github.com/giantswarm/azure-operator/service/controller/v13/encrypter"
	"github.com/giantswarm/azure-operator/service/controller/v13/key"
)

type baseExtension struct {
	azure        setting.Azure
	azureConfig  client.AzureClientSetConfig
	calicoCIDR   string
	clusterCerts certs.Cluster
	customObject providerv1alpha1.AzureConfig
	encrypter    encrypter.Interface
	vnetCIDR     string
}

func (e *baseExtension) templateData(certFiles certs.Files) templateData {
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
			AADClientID:                 e.azureConfig.ClientID,
			AADClientSecret:             e.azureConfig.ClientSecret,
			EnvironmentName:             e.azure.EnvironmentName,
			Location:                    e.azure.Location,
			PrimaryScaleSetName:         key.WorkerVMSSName(e.customObject),
			ResourceGroup:               key.ResourceGroupName(e.customObject),
			RouteTableName:              key.RouteTableName(e.customObject),
			SecurityGroupName:           key.WorkerSecurityGroupName(e.customObject),
			SubnetName:                  key.WorkerSubnetName(e.customObject),
			SubscriptionID:              e.azureConfig.SubscriptionID,
			TenantID:                    e.azureConfig.TenantID,
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
