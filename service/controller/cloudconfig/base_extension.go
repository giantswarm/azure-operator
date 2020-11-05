package cloudconfig

import (
	"github.com/Azure/go-autorest/autorest/azure/auth"
	providerv1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/certs/v3/pkg/certs"
	capzexpv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"

	"github.com/giantswarm/azure-operator/v5/service/controller/encrypter"
	"github.com/giantswarm/azure-operator/v5/service/controller/key"
	"github.com/giantswarm/azure-operator/v5/service/controller/setting"
)

type baseExtension struct {
	azure                        setting.Azure
	azureClientCredentialsConfig auth.ClientCredentialsConfig
	azureMachinePool             *capzexpv1alpha3.AzureMachinePool
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

	primaryScaleSetName := key.WorkerVMSSName(e.customObject)
	if e.azureMachinePool != nil {
		primaryScaleSetName = key.NodePoolVMSSName(e.azureMachinePool)
	}

	subnetName := key.WorkerSubnetName(e.customObject)
	if e.azureMachinePool != nil {
		subnetName = e.azureMachinePool.Name
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
			PrimaryScaleSetName:         primaryScaleSetName,
			ResourceGroup:               key.ResourceGroupName(e.customObject),
			RouteTableName:              key.RouteTableName(e.customObject),
			SecurityGroupName:           key.WorkerSecurityGroupName(e.customObject),
			SubnetName:                  subnetName,
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
