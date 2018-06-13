package cloudconfig

import (
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/controller/setting"
	"github.com/giantswarm/azure-operator/service/controller/v2/key"
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

type cloudProviderConfFileVMType string

type cloudProviderConfFileParams struct {
	Cloud               string
	Location            string
	PrimaryScaleSetName string
	ResourceGroup       string
	RouteTableName      string
	SecurityGroupName   string
	SubnetName          string
	SubscriptionID      string
	TenantID            string
	VnetName            string
}

func newCloudProviderConfFileParams(azure setting.Azure, azureConfig client.AzureClientSetConfig, obj providerv1alpha1.AzureConfig) cloudProviderConfFileParams {
	return cloudProviderConfFileParams{
		Cloud:               azure.Cloud,
		Location:            azure.Location,
		PrimaryScaleSetName: key.WorkerVMSSName(obj),
		ResourceGroup:       key.ResourceGroupName(obj),
		RouteTableName:      key.RouteTableName(obj),
		SecurityGroupName:   key.WorkerSecurityGroupName(obj),
		SubnetName:          key.WorkerSubnetName(obj),
		SubscriptionID:      azureConfig.SubscriptionID,
		TenantID:            azureConfig.TenantID,
		VnetName:            key.VnetName(obj),
	}
}

type diskParams struct {
	DiskName string
}

type ingressLBFileParams struct {
	ClusterDNSDomain string
}

func newIngressLBFileParams(obj providerv1alpha1.AzureConfig) ingressLBFileParams {
	return ingressLBFileParams{
		ClusterDNSDomain: key.ClusterDNSDomain(obj),
	}
}
