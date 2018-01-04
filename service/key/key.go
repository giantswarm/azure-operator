package key

import (
	"fmt"
	"strings"

	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
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
func AzureCloudType(customObject providerv1alpha1.AzureConfig) string {
	// TODO: For now only public cloud supported.
	return defaultAzureCloudType
}

// CalicoSubnetCidr returns subnet for Calico in CIDR notation format.
func CalicoSubnetCidr(customObject providerv1alpha1.AzureConfig) string {
	return fmt.Sprintf("%s/%d", customObject.Spec.Cluster.Calico.Subnet, customObject.Spec.Cluster.Calico.CIDR)
}

// ClusterCustomer returns the customer ID for this cluster.
func ClusterCustomer(customObject providerv1alpha1.AzureConfig) string {
	return customObject.Spec.Cluster.Customer.ID
}

// ClusterID returns the unique ID for this cluster.
func ClusterID(customObject providerv1alpha1.AzureConfig) string {
	return customObject.Spec.Cluster.ID
}

// DNSZoneAPI returns api parent DNS zone domain name.
func DNSZoneAPI(customObject providerv1alpha1.AzureConfig) string {
	return customObject.Spec.Azure.DNSZones.API.Name
}

// DNSZoneEtcd returns etcd parent DNS zone domain name.
// zone should be created in.
func DNSZoneEtcd(customObject providerv1alpha1.AzureConfig) string {
	return customObject.Spec.Azure.DNSZones.Etcd.Name
}

// DNSZoneIngress returns ingress parent DNS zone domain name.
func DNSZoneIngress(customObject providerv1alpha1.AzureConfig) string {
	return customObject.Spec.Azure.DNSZones.Ingress.Name
}

// DNSZonePrefixAPI returns relative name of the api DNS zone.
func DNSZonePrefixAPI(customObject providerv1alpha1.AzureConfig) string {
	return fmt.Sprintf("%s.k8s", ClusterID(customObject))
}

// DNSZonePrefixEtcd returns relative name of the etcd DNS zone.
func DNSZonePrefixEtcd(customObject providerv1alpha1.AzureConfig) string {
	return fmt.Sprintf("%s.k8s", ClusterID(customObject))
}

// DNSZonePrefixIngress returns relative name of the ingress DNS zone.
func DNSZonePrefixIngress(customObject providerv1alpha1.AzureConfig) string {
	return fmt.Sprintf("%s.k8s", ClusterID(customObject))
}

// DNSZoneResourceGroupAPI returns resource group name of the API
// parent DNS zone.
func DNSZoneResourceGroupAPI(customObject providerv1alpha1.AzureConfig) string {
	return customObject.Spec.Azure.DNSZones.API.ResourceGroup
}

// DNSZoneResourceGroupEtcd returns resource group name of the etcd
// parent DNS zone.
func DNSZoneResourceGroupEtcd(customObject providerv1alpha1.AzureConfig) string {
	return customObject.Spec.Azure.DNSZones.Etcd.ResourceGroup
}

// DNSZoneResourceGroupIngress returns resource group name of the ingress
// parent DNS zone.
func DNSZoneResourceGroupIngress(customObject providerv1alpha1.AzureConfig) string {
	return customObject.Spec.Azure.DNSZones.Ingress.ResourceGroup
}

// KeyVaultName returns the Azure Key Vault name for this cluster.
func KeyVaultName(customObject providerv1alpha1.AzureConfig) string {
	return ClusterID(customObject) + "-vault"
}

// KeyVaultKey takes a certificate file path and returns KeyVault key under
// which the certificate will be stored.
func KeyVaultKey(certPath string) string {
	k := strings.TrimPrefix(certPath, "/")
	k = strings.Replace(k, "/", "-", -1)
	k = strings.Replace(k, ".", "-", -1)
	return k
}

// MasterSecurityGroupName returns name of the security group attached to master subnet.
func MasterSecurityGroupName(customObject providerv1alpha1.AzureConfig) string {
	return fmt.Sprintf("%s-%s", ClusterID(customObject), masterSecurityGroupSuffix)
}

// WorkerSecurityGroupName returns name of the security group attached to worker subnet.
func WorkerSecurityGroupName(customObject providerv1alpha1.AzureConfig) string {
	return fmt.Sprintf("%s-%s", ClusterID(customObject), workerSecurityGroupSuffix)
}

// MasterSubnetName returns name of the master subnet.
func MasterSubnetName(customObject providerv1alpha1.AzureConfig) string {
	return fmt.Sprintf("%s-%s-%s", ClusterID(customObject), virtualNetworkSuffix, masterSubnetSuffix)
}

// WorkerSubnetName returns name of the worker subnet.
func WorkerSubnetName(customObject providerv1alpha1.AzureConfig) string {
	return fmt.Sprintf("%s-%s-%s", ClusterID(customObject), virtualNetworkSuffix, workerSubnetSuffix)
}

// Location returns the physical location where the Resource Group is deployed.
func Location(customObject providerv1alpha1.AzureConfig) string {
	return customObject.Spec.Azure.Location
}

// ResourceGroupName returns name of the resource group for this cluster.
func ResourceGroupName(customObject providerv1alpha1.AzureConfig) string {
	return ClusterID(customObject)
}

// RouteTableName returns name of the route table for this cluster.
func RouteTableName(customObject providerv1alpha1.AzureConfig) string {
	return fmt.Sprintf("%s-%s", ClusterID(customObject), routeTableSuffix)
}

func ToCustomObject(v interface{}) (providerv1alpha1.AzureConfig, error) {
	if v == nil {
		return providerv1alpha1.AzureConfig{}, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", &providerv1alpha1.AzureConfig{}, v)
	}

	customObjectPointer, ok := v.(*providerv1alpha1.AzureConfig)
	if !ok {
		return providerv1alpha1.AzureConfig{}, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", &providerv1alpha1.AzureConfig{}, v)
	}
	customObject := *customObjectPointer

	return customObject, nil
}

// VnetName returns name of the virtual network.
func VnetName(customObject providerv1alpha1.AzureConfig) string {
	return fmt.Sprintf("%s-%s", ClusterID(customObject), virtualNetworkSuffix)
}
