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

// DNSRecordPrefixAPI returns DNS A record relative name for API.
func DNSRecordPrefixAPI(customObject azuretpr.CustomObject) string {
	return "api"
}

// DNSRecordPrefixEtcd returns DNS A record relative name for etcd.
func DNSRecordPrefixEtcd(customObject azuretpr.CustomObject) string {
	return "etcd"
}

// DNSRecordPrefixIngress returns DNS A record relative name for ingress.
func DNSRecordPrefixIngress(customObject azuretpr.CustomObject) string {
	return "ingress"
}

// DNSRecordPrefixIngressWildcard returns DNS CNAME record relative name for
// ingress wildcard domain.
func DNSRecordPrefixIngressWildcard(customObject azuretpr.CustomObject) string {
	return "*"
}

// DNSZoneAPI returns api parent DNS zone domain name.
func DNSZoneAPI(customObject azuretpr.CustomObject) string {
	return customObject.Spec.Azure.DNSZones.API
}

// DNSZoneEtcd returns etcd parent DNS zone domain name.
// zone should be created in.
func DNSZoneEtcd(customObject azuretpr.CustomObject) string {
	return customObject.Spec.Azure.DNSZones.Etcd
}

// DNSZoneIngress returns ingress parent DNS zone domain name.
func DNSZoneIngress(customObject azuretpr.CustomObject) string {
	return customObject.Spec.Azure.DNSZones.Ingress
}

// DNSZonePrefixAPI returns relative name of the api DNS zone.
func DNSZonePrefixAPI(customObject azuretpr.CustomObject) string {
	return fmt.Sprintf("%s.k8s", ClusterID(customObject))
}

// DNSZonePrefixEtcd returns relative name of the etcd DNS zone.
func DNSZonePrefixEtcd(customObject azuretpr.CustomObject) string {
	return fmt.Sprintf("%s.k8s", ClusterID(customObject))
}

// DNSZonePrefixIngress returns relative name of the ingress DNS zone.
func DNSZonePrefixIngress(customObject azuretpr.CustomObject) string {
	return fmt.Sprintf("%s.k8s", ClusterID(customObject))
}

// DNSZoneResourceGroupAPI returns resource group name of the API
// parent DNS zone.
func DNSZoneResourceGroupAPI(customObject azuretpr.CustomObject) string {
	return HostClusterResourceGroupName(customObject)
}

// DNSZoneResourceGroupEtcd returns resource group name of the etcd
// parent DNS zone.
func DNSZoneResourceGroupEtcd(customObject azuretpr.CustomObject) string {
	return HostClusterResourceGroupName(customObject)
}

// DNSZoneResourceGroupIngress returns resource group name of the ingress
// parent DNS zone.
func DNSZoneResourceGroupIngress(customObject azuretpr.CustomObject) string {
	return HostClusterResourceGroupName(customObject)
}

// HostClusterResourceGroupName returns name of the resource group for the host cluster.
func HostClusterResourceGroupName(customObject azuretpr.CustomObject) string {
	return customObject.Spec.Azure.HostCluster.ResourceGroup
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
