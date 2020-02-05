package key

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Azure/go-autorest/autorest/to"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/azure-operator/service/controller/v13/templates/ignition"
	"github.com/giantswarm/microerror"
)

const (
	clusterTagName      = "GiantSwarmCluster"
	installationTagName = "GiantSwarmInstallation"
	organizationTagName = "GiantSwarmOrganization"
	VmssDeploymentName  = "cluster-vmss-template"

	blobContainerName = "ignition"
	// cloudConfigVersion is used in blob object ignition name
	cloudConfigVersion        = "v4.7.0"
	storageAccountSuffix      = "gssa"
	routeTableSuffix          = "RouteTable"
	masterSecurityGroupSuffix = "MasterSecurityGroup"
	workerSecurityGroupSuffix = "WorkerSecurityGroup"
	masterSubnetSuffix        = "MasterSubnet"
	workerSubnetSuffix        = "WorkerSubnet"
	prefixMaster              = "master"
	prefixWorker              = "worker"
	virtualNetworkSuffix      = "VirtualNetwork"
	vpnGatewaySubnet          = "GatewaySubnet"
	vpnGatewaySuffix          = "VPNGateway"

	TemplateContentVersion = "1.0.0.0"

	AnnotationEtcdDomain        = "giantswarm.io/etcd-domain"
	AnnotationPrometheusCluster = "giantswarm.io/prometheus-cluster"

	LabelApp           = "app"
	LabelCluster       = "giantswarm.io/cluster"
	LabelCustomer      = "customer"
	LabelManagedBy     = "giantswarm.io/managed-by"
	LabelOrganization  = "giantswarm.io/organization"
	LabelVersionBundle = "giantswarm.io/version-bundle"

	LegacyLabelCluster = "cluster"

	CertificateEncryptionNamespace = "default"
	CertificateEncryptionKeyName   = "encryptionkey"
	CertificateEncryptionIVName    = "encryptioniv"
)

const (
	ClusterIDLabel = "giantswarm.io/cluster"
)

func AdminUsername(customObject providerv1alpha1.AzureConfig) string {
	users := customObject.Spec.Cluster.Kubernetes.SSH.UserList
	// We don't want panics when someone is doing something nasty.
	if len(users) == 0 {
		return ""
	}
	return users[0].Name
}

func AdminSSHKeyData(customObject providerv1alpha1.AzureConfig) string {
	users := customObject.Spec.Cluster.Kubernetes.SSH.UserList
	// We don't want panics when someone is doing something nasty.
	if len(users) == 0 {
		return ""
	}
	return users[0].PublicKey
}

func APISecurePort(customObject providerv1alpha1.AzureConfig) int {
	return customObject.Spec.Cluster.Kubernetes.API.SecurePort
}

// ARMTemplateURI returns URI to a resource's template file.
// The returned URI point to a file hosted on github in this repository
// under a controller resource's template folder at:
//
// service/controller/v13/resource/<resource>/template/<template>
//
// version refers to a branch or commit.
// resource refers to directory name of the resource.
// template refers to filename under resource's template folder.
//
// e.g. ARMTemplateURI("master", "deployment", "main.json")
func ARMTemplateURI(version, resource, template string) string {
	return fmt.Sprintf("https://raw.githubusercontent.com/giantswarm/azure-operator/%s/service/controller/v13/resource/%s/template/%s", version, resource, template)
}

func BlobContainerName() string {
	return blobContainerName
}

func BlobName(customObject providerv1alpha1.AzureConfig, role string) string {
	return fmt.Sprintf("%s-%s-%s", VersionBundleVersion(customObject), cloudConfigVersion, role)
}

func CertificateEncryptionSecretName(customObject providerv1alpha1.AzureConfig) string {
	return fmt.Sprintf("%s-certificate-encryption", customObject.Spec.Cluster.ID)
}

func CloudConfigSmallTemplates() []string {
	return []string{
		ignition.Small,
	}
}

func ClusterAPIEndpoint(customObject providerv1alpha1.AzureConfig) string {
	return customObject.Spec.Cluster.Kubernetes.API.Domain
}

// ClusterCustomer returns the customer ID for this cluster.
func ClusterCustomer(customObject providerv1alpha1.AzureConfig) string {
	return customObject.Spec.Cluster.Customer.ID
}

// ClusterDNSDomain returns cluster DNS domain.
func ClusterDNSDomain(customObject providerv1alpha1.AzureConfig) string {
	return fmt.Sprintf("%s.%s", DNSZonePrefixAPI(customObject), DNSZoneAPI(customObject))
}

func ClusterEtcdDomain(customObject providerv1alpha1.AzureConfig) string {
	return fmt.Sprintf("%s:%d", customObject.Spec.Cluster.Etcd.Domain, customObject.Spec.Cluster.Etcd.Port)
}

// ClusterID returns the unique ID for this cluster.
func ClusterID(customObject providerv1alpha1.AzureConfig) string {
	return customObject.Spec.Cluster.ID
}

// ClusterNamespace returns the cluster Namespace for this cluster.
func ClusterNamespace(customObject providerv1alpha1.AzureConfig) string {
	return customObject.Spec.Cluster.ID
}

// ClusterOrganization returns the org name from the custom object.
// It uses ClusterCustomer until this field is renamed in the custom object.
func ClusterOrganization(customObject providerv1alpha1.AzureConfig) string {
	return ClusterCustomer(customObject)
}

// ClusterTags returns a map with the resource tags for this cluster.
func ClusterTags(customObject providerv1alpha1.AzureConfig, installationName string) map[string]*string {
	tags := map[string]*string{
		clusterTagName:      to.StringPtr(ClusterID(customObject)),
		installationTagName: to.StringPtr(installationName),
		organizationTagName: to.StringPtr(ClusterOrganization(customObject)),
	}

	return tags
}

// CredentialName returns name of the credential secret.
func CredentialName(customObject providerv1alpha1.AzureConfig) string {
	return customObject.Spec.Azure.CredentialSecret.Name
}

// CredentialNamespace returns namespace of the credential secret.
func CredentialNamespace(customObject providerv1alpha1.AzureConfig) string {
	return customObject.Spec.Azure.CredentialSecret.Namespace
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

func DNSZones(customObject providerv1alpha1.AzureConfig) providerv1alpha1.AzureConfigSpecAzureDNSZones {
	return customObject.Spec.Azure.DNSZones
}

func IsDeleted(customObject providerv1alpha1.AzureConfig) bool {
	return customObject.GetDeletionTimestamp() != nil
}

func IsFinalProvisioningState(s string) bool {
	return IsFailedProvisioningState(s) || IsSucceededProvisioningState(s)
}

func IsFailedProvisioningState(s string) bool {
	if s == "Failed" {
		return true
	}
	if s == "Canceled" {
		return true
	}

	return false
}

func IsSucceededProvisioningState(s string) bool {
	if s == "Succeeded" {
		return true
	}

	return false
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

func MasterInstanceName(customObject providerv1alpha1.AzureConfig, instanceID string) (string, error) {
	idB36, err := vmssInstanceIDBase36(instanceID)
	if err != nil {
		return "", microerror.Mask(err)
	}

	return fmt.Sprintf("%s-master-%06s", ClusterID(customObject), idB36), nil
}

// MasterNICName returns name of the master NIC.
func MasterNICName(customObject providerv1alpha1.AzureConfig) string {
	return fmt.Sprintf("%s-Master-1-NIC", ClusterID(customObject))
}

func MasterVMSSName(customObject providerv1alpha1.AzureConfig) string {
	return fmt.Sprintf("%s-master", ClusterID(customObject))
}

func PrefixMaster() string {
	return prefixMaster
}

func PrefixWorker() string {
	return prefixWorker
}

// ResourceGroupName returns name of the resource group for this cluster.
func ResourceGroupName(customObject providerv1alpha1.AzureConfig) string {
	return ClusterID(customObject)
}

// RouteTableName returns name of the route table for this cluster.
func RouteTableName(customObject providerv1alpha1.AzureConfig) string {
	return fmt.Sprintf("%s-%s", ClusterID(customObject), routeTableSuffix)
}

// AvailabilityZones returns the availability zones where the cluster will be created.
func AvailabilityZones(customObject providerv1alpha1.AzureConfig) []int {
	if customObject.Spec.Azure.AvailabilityZones == nil {
		return []int{}
	}

	return customObject.Spec.Azure.AvailabilityZones
}

func StorageAccountName(customObject providerv1alpha1.AzureConfig) string {
	// In integration tests we use hyphens which are not allowed. We also
	// need to keep the name globaly unique and within 24 character limit.
	//
	//	See https://docs.microsoft.com/en-us/azure/architecture/best-practices/naming-conventions#storage
	//
	storageAccountName := fmt.Sprintf("%s%s", storageAccountSuffix, ClusterID(customObject))
	return strings.Replace(storageAccountName, "-", "", -1)
}

func ToClusterEndpoint(v interface{}) (string, error) {
	customObject, err := ToCustomObject(v)
	if err != nil {
		return "", microerror.Mask(err)
	}

	return customObject.Spec.Cluster.Kubernetes.API.Domain, nil
}

func ToClusterID(v interface{}) (string, error) {
	customObject, err := ToCustomObject(v)
	if err != nil {
		return "", microerror.Mask(err)
	}

	return ClusterID(customObject), nil
}

func ToClusterStatus(v interface{}) (providerv1alpha1.StatusCluster, error) {
	customObject, err := ToCustomObject(v)
	if err != nil {
		return providerv1alpha1.StatusCluster{}, microerror.Mask(err)
	}

	return customObject.Status.Cluster, nil
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

func ToKeyValue(m map[string]interface{}) (interface{}, error) {
	v, ok := m["value"]
	if !ok {
		return "", microerror.Mask(missingOutputValueError)
	}

	return v, nil
}

func ToMap(v interface{}) (map[string]interface{}, error) {
	m, ok := v.(map[string]interface{})
	if !ok {
		return nil, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", map[string]interface{}{}, v)
	}

	return m, nil
}

func ToNodeCount(v interface{}) (int, error) {
	customObject, err := ToCustomObject(v)
	if err != nil {
		return 0, microerror.Mask(err)
	}

	nodeCount := len(customObject.Spec.Azure.Masters) + len(customObject.Spec.Azure.Workers)

	return nodeCount, nil
}

// ToParameters merges the input maps and converts the result into the
// structure used by the Azure API. Note that the order of inputs is relevant.
// Default parameters should be given first. Data of the following maps will
// overwrite eventual data of preceeding maps. This mechanism is used for e.g.
// setting the initialProvisioning parameter accordingly to the cluster's state.
func ToParameters(list ...map[string]interface{}) map[string]interface{} {
	allParams := map[string]interface{}{}

	for _, l := range list {
		for key, val := range l {
			allParams[key] = struct {
				Value interface{}
			}{
				Value: val,
			}
		}
	}

	return allParams
}

func ToString(v interface{}) (string, error) {
	s, ok := v.(string)
	if !ok {
		return "", microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", "", v)
	}

	return s, nil
}

func ToStringMap(v interface{}) (map[string]string, error) {
	m, err := ToMap(v)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	stringMap := map[string]string{}
	for k, v := range m {
		s, err := ToString(v)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		stringMap[k] = s
	}

	return stringMap, nil
}

func ToVersionBundleVersion(v interface{}) (string, error) {
	customObject, err := ToCustomObject(v)
	if err != nil {
		return "", microerror.Mask(err)
	}

	return VersionBundleVersion(customObject), nil
}

func VersionBundleVersion(customObject providerv1alpha1.AzureConfig) string {
	return customObject.Spec.VersionBundle.Version
}

// VnetName returns name of the virtual network.
func VnetName(customObject providerv1alpha1.AzureConfig) string {
	return fmt.Sprintf("%s-%s", ClusterID(customObject), virtualNetworkSuffix)
}

func VnetCIDR(customObject providerv1alpha1.AzureConfig) string {
	return customObject.Spec.Azure.VirtualNetwork.CIDR
}

// VNetGatewaySubnetName returns the name of the subnet for the vpn gateway.
func VNetGatewaySubnetName() string {
	return vpnGatewaySubnet
}

func VNetID(customObject providerv1alpha1.AzureConfig, subscriptionID string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/%s", subscriptionID, ResourceGroupName(customObject), VnetName(customObject))
}

// VPNGatewayName returns name of the vpn gateway.
func VPNGatewayName(customObject providerv1alpha1.AzureConfig) string {
	return fmt.Sprintf("%s-%s", ClusterID(customObject), vpnGatewaySuffix)
}

func WorkerInstanceName(customObject providerv1alpha1.AzureConfig, instanceID string) (string, error) {
	idB36, err := vmssInstanceIDBase36(instanceID)
	if err != nil {
		return "", microerror.Mask(err)
	}

	return fmt.Sprintf("%s-worker-%06s", ClusterID(customObject), idB36), nil
}

func WorkerVMSSName(customObject providerv1alpha1.AzureConfig) string {
	return fmt.Sprintf("%s-worker", ClusterID(customObject))
}

func vmssInstanceIDBase36(instanceID string) (string, error) {
	i, err := strconv.ParseUint(instanceID, 10, 64)
	if err != nil {
		// TODO Avoid panic call below if feasible.
		//
		//	See https://github.com/giantswarm/giantswarm/issues/4674
		//

		// This must be an int according to the documentation linked below.
		//
		// We are panicking here to make the API nice. If this is not
		// an int there is nothing we can really do and we need to
		// redesign `instance` resource.
		//
		//	https://docs.microsoft.com/en-us/azure/virtual-machine-scale-sets/virtual-machine-scale-sets-instance-ids#scale-set-vm-computer-name
		//
		return "", microerror.Maskf(executionFailedError, "expected VMSS instanceID to be a positive integer number but got %#q", instanceID)
	}

	return strconv.FormatUint(i, 36), nil
}
