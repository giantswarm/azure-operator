package key

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Azure/go-autorest/autorest/to"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	releasev1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/release/v1alpha1"
	k8scloudconfig "github.com/giantswarm/k8scloudconfig/v6/pkg/template"
	"github.com/giantswarm/microerror"
	"sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"

	"github.com/giantswarm/azure-operator/v4/service/controller/templates/ignition"
)

const (
	clusterTagName            = "GiantSwarmCluster"
	installationTagName       = "GiantSwarmInstallation"
	organizationTagName       = "GiantSwarmOrganization"
	MastersVmssDeploymentName = "masters-vmss-template"
	WorkersVmssDeploymentName = "workers-vmss-template"

	blobContainerName = "ignition"
	// cloudConfigVersion is used in blob object ignition name
	cloudConfigVersion        = "v6.0.0"
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

	AnnotationEtcdDomain        = "giantswarm.io/etcd-domain"
	AnnotationPrometheusCluster = "giantswarm.io/prometheus-cluster"

	LabelApp             = "app"
	LabelCluster         = "giantswarm.io/cluster"
	LabelCustomer        = "customer"
	LabelManagedBy       = "giantswarm.io/managed-by"
	LabelOperatorVersion = "azure-operator.giantswarm.io/version"
	LabelOrganization    = "giantswarm.io/organization"

	LegacyLabelCluster = "cluster"

	CertificateEncryptionNamespace = "default"
	CertificateEncryptionKeyName   = "encryptionkey"
	CertificateEncryptionIVName    = "encryptioniv"

	ContainerLinuxComponentName = "containerlinux"

	OrganizationSecretsLabelSelector = "app=credentiald" // nolint:gosec
)

// Container image versions for k8scloudconfig.
const (
	// k8s-api-healthz version.
	kubernetesAPIHealthzVersion = "0999549a4c334b646288d08bd2c781c6aae2e12f"

	// k8s-setup-network-environment.
	kubernetesNetworkSetupDocker = "1f4ffc52095ac368847ce3428ea99b257003d9b9"
)

const (
	ClusterIDLabel = "giantswarm.io/cluster"
)

var (
	LocationsThatDontSupportAZs = []string{
		"germanywestcentral",
	}
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

func AzureConfigNetworkCIDR(customObject providerv1alpha1.AzureConfig) string {
	return customObject.Spec.Azure.VirtualNetwork.CIDR
}

func ToAzureMachinePool(v interface{}) (v1alpha3.AzureMachinePool, error) {
	if v == nil {
		return v1alpha3.AzureMachinePool{}, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", &providerv1alpha1.AzureConfig{}, v)
	}

	customObjectPointer, ok := v.(*v1alpha3.AzureMachinePool)
	if !ok {
		return v1alpha3.AzureMachinePool{}, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", &providerv1alpha1.AzureConfig{}, v)
	}
	customObject := *customObjectPointer

	return customObject, nil
}

func BlobContainerName() string {
	return blobContainerName
}

func BlobName(customObject providerv1alpha1.AzureConfig, role string) string {
	return fmt.Sprintf("%s-%s-%s", OperatorVersion(customObject), cloudConfigVersion, role)
}

func WorkerBlobName(operatorVersion string) string {
	return fmt.Sprintf("%s-%s-%s", operatorVersion, cloudConfigVersion, prefixWorker)
}

func CalicoCIDR(customObject providerv1alpha1.AzureConfig) string {
	return customObject.Spec.Azure.VirtualNetwork.CalicoSubnetCIDR
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

func ClusterBaseDomain(customObject providerv1alpha1.AzureConfig) string {
	apiDomainComponents := strings.Split(ClusterAPIEndpoint(customObject), ".")
	if len(apiDomainComponents) > 2 {
		// Drop `api` prefix component.
		apiDomainComponents = apiDomainComponents[1:]
	}

	return strings.Join(apiDomainComponents, ".")
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

// ComponentVersion returns the version of the given component in the Release.
func ComponentVersion(release releasev1alpha1.Release, componentName string) (string, error) {
	for _, component := range release.Spec.Components {
		if component.Name == componentName {
			return component.Version, nil
		}
	}
	return "", microerror.Maskf(notFoundError, "version for component %#v not found on release %#v", componentName, release.Name)
}

// CredentialName returns name of the credential secret.
func CredentialName(customObject providerv1alpha1.AzureConfig) string {
	return customObject.Spec.Azure.CredentialSecret.Name
}

// CredentialNamespace returns namespace of the credential secret.
func CredentialNamespace(customObject providerv1alpha1.AzureConfig) string {
	return customObject.Spec.Azure.CredentialSecret.Namespace
}

func DefaultVersions() k8scloudconfig.Versions {
	return k8scloudconfig.Versions{
		KubernetesAPIHealthz:         kubernetesAPIHealthzVersion,
		KubernetesNetworkSetupDocker: kubernetesNetworkSetupDocker,
	}
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
	return s == "Succeeded"
}

func MachinePoolOperatorVersion(cr v1alpha3.AzureMachinePool) string {
	return cr.GetLabels()[LabelOperatorVersion]
}

// MasterSecurityGroupName returns name of the security group attached to master subnet.
func MasterSecurityGroupName(customObject providerv1alpha1.AzureConfig) string {
	return fmt.Sprintf("%s-%s", ClusterID(customObject), masterSecurityGroupSuffix)
}

// OSVersion returns the version of the operating system.
func OSVersion(release releasev1alpha1.Release) (string, error) {
	v, err := ComponentVersion(release, ContainerLinuxComponentName)
	if err != nil {
		return "", microerror.Mask(err)
	}
	return v, nil
}

// WorkerSecurityGroupName returns name of the security group attached to worker subnet.
func WorkerSecurityGroupName(customObject providerv1alpha1.AzureConfig) string {
	return fmt.Sprintf("%s-%s", ClusterID(customObject), workerSecurityGroupSuffix)
}

// MasterSubnetName returns name of the master subnet.
func MasterSubnetName(customObject providerv1alpha1.AzureConfig) string {
	return fmt.Sprintf("%s-%s-%s", ClusterID(customObject), virtualNetworkSuffix, masterSubnetSuffix)
}

func MastersSubnetCIDR(customObject providerv1alpha1.AzureConfig) string {
	return customObject.Spec.Azure.VirtualNetwork.MasterSubnetCIDR
}

// WorkerCount returns the desired number of workers.
func WorkerCount(customObject providerv1alpha1.AzureConfig) int {
	return len(customObject.Spec.Azure.Workers)
}

// WorkerSubnetName returns name of the worker subnet.
func WorkerSubnetName(customObject providerv1alpha1.AzureConfig) string {
	return fmt.Sprintf("%s-%s-%s", ClusterID(customObject), virtualNetworkSuffix, workerSubnetSuffix)
}

func MasterInstanceName(customObject providerv1alpha1.AzureConfig, instanceID string) string {
	idB36, err := vmssInstanceIDBase36(instanceID)
	if err != nil {
		panic(err)
	}

	return fmt.Sprintf("%s-master-%s-%06s", ClusterID(customObject), ClusterID(customObject), idB36)
}

// MasterNICName returns name of the master NIC.
func MasterNICName(customObject providerv1alpha1.AzureConfig) string {
	return fmt.Sprintf("%s-Master-1-NIC", ClusterID(customObject))
}

func LegacyMasterVMSSName(customObject providerv1alpha1.AzureConfig) string {
	return fmt.Sprintf("%s-master", ClusterID(customObject))
}

func MasterVMSSName(customObject providerv1alpha1.AzureConfig) string {
	return fmt.Sprintf("%s-master-%s", ClusterID(customObject), ClusterID(customObject))
}

func OperatorVersion(cr providerv1alpha1.AzureConfig) string {
	return cr.GetLabels()[LabelOperatorVersion]
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
func AvailabilityZones(customObject providerv1alpha1.AzureConfig, location string) []int {
	for _, l := range LocationsThatDontSupportAZs {
		if l == location {
			return []int{}
		}
	}

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
	cr, err := ToCustomResource(v)
	if err != nil {
		return "", microerror.Mask(err)
	}

	return cr.Spec.Cluster.Kubernetes.API.Domain, nil
}

func ToClusterID(v interface{}) (string, error) {
	cr, err := ToCustomResource(v)
	if err != nil {
		return "", microerror.Mask(err)
	}

	return ClusterID(cr), nil
}

func ToClusterStatus(v interface{}) (providerv1alpha1.StatusCluster, error) {
	cr, err := ToCustomResource(v)
	if err != nil {
		return providerv1alpha1.StatusCluster{}, microerror.Mask(err)
	}

	return cr.Status.Cluster, nil
}

func ToCustomResource(v interface{}) (providerv1alpha1.AzureConfig, error) {
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
	cr, err := ToCustomResource(v)
	if err != nil {
		return 0, microerror.Mask(err)
	}

	nodeCount := len(cr.Spec.Azure.Masters) + len(cr.Spec.Azure.Workers)

	return nodeCount, nil
}

func ToOperatorVersion(v interface{}) (string, error) {
	customObject, err := ToCustomResource(v)
	if err != nil {
		return "", microerror.Mask(err)
	}

	return OperatorVersion(customObject), nil
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

// VPNGatewayName returns name of the vpn gateway.
func VPNGatewayName(customObject providerv1alpha1.AzureConfig) string {
	return fmt.Sprintf("%s-%s", ClusterID(customObject), vpnGatewaySuffix)
}

func LegacyWorkerInstanceName(customObject providerv1alpha1.AzureConfig, instanceID string) string {
	idB36, err := vmssInstanceIDBase36(instanceID)
	if err != nil {
		panic(err)
	}

	return fmt.Sprintf("%s-worker-%06s", ClusterID(customObject), idB36)
}

func WorkerInstanceName(customObject providerv1alpha1.AzureConfig, instanceID string) string {
	idB36, err := vmssInstanceIDBase36(instanceID)
	if err != nil {
		panic(err)
	}

	return fmt.Sprintf("%s-worker-%s-%06s", ClusterID(customObject), ClusterID(customObject), idB36)
}

func LegacyWorkerVMSSName(customObject providerv1alpha1.AzureConfig) string {
	return fmt.Sprintf("%s-worker", ClusterID(customObject))
}

func WorkerVMSSName(customObject providerv1alpha1.AzureConfig) string {
	return fmt.Sprintf("%s-worker-%s", ClusterID(customObject), ClusterID(customObject))
}

func WorkersSubnetCIDR(customObject providerv1alpha1.AzureConfig) string {
	return customObject.Spec.Azure.VirtualNetwork.WorkerSubnetCIDR
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
