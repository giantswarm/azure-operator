package cloudconfig

import (
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/controller/setting"
	"github.com/giantswarm/azure-operator/service/controller/v1/key"
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

const (
	vmTypeStandard cloudProviderConfFileVMType = "standard"
	vmTypeVMSS     cloudProviderConfFileVMType = "vmss"
)

type cloudProviderConfFileParams struct {
	Location            string
	PrimaryScaleSetName string
	ResourceGroup       string
	RouteTableName      string
	SecurityGroupName   string
	SubnetName          string
	SubscriptionID      string
	TenantID            string
	VnetName            string
	VMType              cloudProviderConfFileVMType
}

func newCloudProviderConfFileParams(azure setting.Azure, azureConfig client.AzureConfig, obj providerv1alpha1.AzureConfig, vmType cloudProviderConfFileVMType) cloudProviderConfFileParams {
	return cloudProviderConfFileParams{
		Location:            azure.Location,
		PrimaryScaleSetName: key.WorkerVMSSName(obj),
		ResourceGroup:       key.ResourceGroupName(obj),
		RouteTableName:      key.RouteTableName(obj),
		SecurityGroupName:   key.WorkerSecurityGroupName(obj),
		SubnetName:          key.WorkerSubnetName(obj),
		SubscriptionID:      azureConfig.SubscriptionID,
		TenantID:            azureConfig.TenantID,
		VnetName:            key.VnetName(obj),
		VMType:              vmType,
	}
}

type diskParams struct {
	DiskName string
}
