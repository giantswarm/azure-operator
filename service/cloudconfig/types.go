package cloudconfig

import (
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/key"
)

type cloudProviderConfFileParams struct {
	AzureCloudType    string
	Location          string
	ResourceGroup     string
	RouteTableName    string
	SecurityGroupName string
	SubnetName        string
	SubscriptionID    string
	TenantID          string
	VnetName          string
}

func newCloudProviderConfFileParams(azureConfig client.AzureConfig, obj providerv1alpha1.AzureConfig) cloudProviderConfFileParams {
	return cloudProviderConfFileParams{
		AzureCloudType:    key.AzureCloudType(obj),
		Location:          key.Location(obj),
		ResourceGroup:     key.ResourceGroupName(obj),
		RouteTableName:    key.RouteTableName(obj),
		SecurityGroupName: key.WorkerSecurityGroupName(obj),
		SubnetName:        key.WorkerSubnetName(obj),
		SubscriptionID:    azureConfig.SubscriptionID,
		TenantID:          azureConfig.TenantID,
		VnetName:          key.VnetName(obj),
	}
}

type getKeyVaultSecretsFileParams struct {
	VaultName string
	Secrets   []keyVaultSecret
}

type keyVaultSecret struct {
	SecretName string
	FileName   string
}

type cloudConfigExtension struct {
	AzureConfig  client.AzureConfig
	CustomObject providerv1alpha1.AzureConfig
}

type masterExtension struct {
	cloudConfigExtension
}

type workerExtension struct {
	cloudConfigExtension
}
