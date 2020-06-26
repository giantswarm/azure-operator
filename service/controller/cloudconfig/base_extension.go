package cloudconfig

import (
	"github.com/Azure/go-autorest/autorest/azure/auth"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/certs"

	"github.com/giantswarm/azure-operator/v4/service/controller/encrypter"
	"github.com/giantswarm/azure-operator/v4/service/controller/setting"
)

type baseExtension struct {
	azure                        setting.Azure
	azureClientCredentialsConfig auth.ClientCredentialsConfig
	calicoCIDR                   string
	cluster                      providerv1alpha1.Cluster
	clusterCerts                 certs.Cluster
	clusterDNSDomain             string
	primaryScaleSetName          string
	resourceGroup                string
	routeTableName               string
	securityGroupName            string
	subnetName                   string
	encrypter                    encrypter.Interface
	subscriptionID               string
	vnetCIDR                     string
	vnetName                     string
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
			Cluster:    e.cluster,
			CalicoCIDR: e.calicoCIDR,
		},
		cloudProviderConfFileParams{
			AADClientID:                 e.azureClientCredentialsConfig.ClientID,
			AADClientSecret:             e.azureClientCredentialsConfig.ClientSecret,
			EnvironmentName:             e.azure.EnvironmentName,
			Location:                    e.azure.Location,
			PrimaryScaleSetName:         e.primaryScaleSetName,
			ResourceGroup:               e.resourceGroup,
			RouteTableName:              e.routeTableName,
			SecurityGroupName:           e.securityGroupName,
			SubnetName:                  e.subnetName,
			SubscriptionID:              e.subscriptionID,
			TenantID:                    e.azureClientCredentialsConfig.TenantID,
			VnetName:                    e.vnetName,
			UseManagedIdentityExtension: e.azure.MSI.Enabled,
		},
		certificateDecrypterUnitParams{
			CertsPaths: certsPaths,
		},
		ingressLBFileParams{
			ClusterDNSDomain: e.clusterDNSDomain,
		},
	}
}
