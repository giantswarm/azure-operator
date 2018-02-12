package cloudconfig

import (
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/key"
)

type calicoAzureFileParams struct {
	Cluster    providerv1alpha1.Cluster
	CalicoCIDR string
}

func newCalicoAzureFileParams(obj providerv1alpha1.AzureConfig) calicoAzureFileParams {
	return calicoAzureFileParams{
		Cluster:    obj.Spec.Cluster,
		CalicoCIDR: key.VnetCalicoSubnetCIDR(obj),
	}
}

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
	Secrets   []getKeyVaultSecretsFileParamsSecret
}

type getKeyVaultSecretsFileParamsSecret struct {
	SecretName string
	FileName   string
}

type diskParams struct {
	DiskName string
}
