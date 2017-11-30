package key

import (
	"fmt"

	"github.com/giantswarm/azuretpr"
	"github.com/giantswarm/certificatetpr"
)

const (
	defaultAzureCloudType     = "AZUREPUBLICCLOUD"
	routeTableSuffix          = "RouteTable"
	masterSecurityGroupSuffix = "MasterSecurityGroup"
	workerSecurityGroupSuffix = "WorkerSecurityGroup"
	masterSubnetSuffix        = "MasterSubnet"
	workerSubnetSuffix        = "WorkerSubnet"
	virtualNetworkSuffix      = "VirtualNetwork"
)

// AzureCloudType returns cloud type.
func AzureCloudType(customObject azuretpr.CustomObject) string {
	// TODO: For now only public cloud supported.
	return defaultAzureCloudType
}

// ClusterCustomer returns the customer ID for this cluster.
func ClusterCustomer(customObject azuretpr.CustomObject) string {
	return customObject.Spec.Cluster.Customer.ID
}

// ClusterID returns the unique ID for this cluster.
func ClusterID(customObject azuretpr.CustomObject) string {
	return customObject.Spec.Cluster.Cluster.ID
}

// KeyVaultName returns the Azure Key Vault name for this cluster.
func KeyVaultName(customObject azuretpr.CustomObject) string {
	return customObject.Spec.Azure.KeyVault.Name
}

// MasterSecurityGroupName returns name of the security group attached to master subnet.
func MasterSecurityGroupName(customObject azuretpr.CustomObject) string {
	return fmt.Sprintf("%s-%s", customObject.Spec.Cluster.Cluster.ID, masterSecurityGroupSuffix)
}

// WorkerSecurityGroupName returns name of the security group attached to worker subnet.
func WorkerSecurityGroupName(customObject azuretpr.CustomObject) string {
	return fmt.Sprintf("%s-%s", customObject.Spec.Cluster.Cluster.ID, workerSecurityGroupSuffix)
}

// MasterSubnetName returns name of the master subnet.
func MasterSubnetName(customObject azuretpr.CustomObject) string {
	return fmt.Sprintf("%s-%s-%s", customObject.Spec.Cluster.Cluster.ID, virtualNetworkSuffix, masterSubnetSuffix)
}

// WorkerSubnetName returns name of the worker subnet.
func WorkerSubnetName(customObject azuretpr.CustomObject) string {
	return fmt.Sprintf("%s-%s-%s", customObject.Spec.Cluster.Cluster.ID, virtualNetworkSuffix, workerSubnetSuffix)
}

// Location returns the physical location where the Resource Group is deployed.
func Location(customObject azuretpr.CustomObject) string {
	return customObject.Spec.Azure.Location
}

// ResourceGroupName returns name of the resource group for this cluster.
func ResourceGroupName(customObject azuretpr.CustomObject) string {
	return customObject.Spec.Cluster.Cluster.ID
}

// RouteTableName returns name of the route table for this cluster.
func RouteTableName(customObject azuretpr.CustomObject) string {
	return fmt.Sprintf("%s-%s", customObject.Spec.Cluster.Cluster.ID, routeTableSuffix)
}

// SecretName returns the name of the Key Vault secret for this certificate
// asset.
func SecretName(clusterComponent certificatetpr.ClusterComponent, assetType certificatetpr.TLSAssetType) string {
	return fmt.Sprintf("%s-%s", clusterComponent, assetType)
}

// VnetName returns name of the virtual network.
func VnetName(customObject azuretpr.CustomObject) string {
	return fmt.Sprintf("%s-%s", customObject.Spec.Cluster.Cluster.ID, virtualNetworkSuffix)
}
