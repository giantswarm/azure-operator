package key

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/Azure/go-autorest/autorest/to"
	apiextensionsannotations "github.com/giantswarm/apiextensions/v3/pkg/annotation"
	"github.com/giantswarm/apiextensions/v3/pkg/apis/provider/v1alpha1"
	providerv1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/provider/v1alpha1"
	apiextensionslabels "github.com/giantswarm/apiextensions/v3/pkg/label"
	k8scloudconfig "github.com/giantswarm/k8scloudconfig/v10/pkg/template"
	"github.com/giantswarm/microerror"
	v1 "k8s.io/api/core/v1"
	capzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	expcapzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	expcapiv1alpha3 "sigs.k8s.io/cluster-api/exp/api/v1alpha3"

	"github.com/giantswarm/azure-operator/v5/pkg/employees"
	"github.com/giantswarm/azure-operator/v5/pkg/label"
	"github.com/giantswarm/azure-operator/v5/service/controller/templates/ignition"
)

const (
	clusterTagName            = "GiantSwarmCluster"
	installationTagName       = "GiantSwarmInstallation"
	organizationTagName       = "GiantSwarmOrganization"
	MastersVmssDeploymentName = "masters-vmss-template"

	// Kept for the sake of compiling old instance resource. It should be removed as soon as
	// instance resource is removed.
	WorkersVmssDeploymentName = "workers-vmss-template"

	CloudConfigSecretKey = "ignitionBlob"
	blobContainerName    = "ignition"

	// cloudConfigVersion is used in blob object ignition name
	cloudConfigVersion        = "v7.0.1"
	storageAccountSuffix      = "gssa"
	routeTableSuffix          = "RouteTable"
	masterSecurityGroupSuffix = "MasterSecurityGroup"
	workerSecurityGroupSuffix = "WorkerSecurityGroup"
	masterSubnetSuffix        = "MasterSubnet"
	workerSubnetSuffix        = "WorkerSubnet"
	masterNatGatewayName      = "masters-nat-gw"
	prefixMaster              = "master"
	prefixWorker              = "worker"
	subnetDeploymentPrefix    = "subnet"
	virtualNetworkSuffix      = "VirtualNetwork"
	vpnGatewaySubnet          = "GatewaySubnet"
	vpnGatewaySuffix          = "VPNGateway"

	AnnotationEtcdDomain        = "giantswarm.io/etcd-domain"
	AnnotationPrometheusCluster = "giantswarm.io/prometheus-cluster"
	AnnotationNodePoolMinSize   = "cluster.k8s.io/cluster-api-autoscaler-node-group-min-size"
	AnnotationNodePoolMaxSize   = "cluster.k8s.io/cluster-api-autoscaler-node-group-max-size"

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

	credentialDefaultNamespace = "giantswarm"
	credentialDefaultName      = "credential-default"
)

// Container image versions for k8scloudconfig.
const (
	// k8s-api-healthz version.
	kubernetesAPIHealthzVersion = "0.1.1"

	// k8s-setup-network-environment.
	kubernetesNetworkSetupDocker = "0.2.0"
)

var (
	instanceIDRegexp = regexp.MustCompile(`-[^-]{6}$`)
)

func APISecurePort(customObject providerv1alpha1.AzureConfig) int {
	return customObject.Spec.Cluster.Kubernetes.API.SecurePort
}

func AzureConfigNetworkCIDR(customObject providerv1alpha1.AzureConfig) string {
	return customObject.Spec.Azure.VirtualNetwork.CIDR
}

func AzureMachineName(getter LabelsGetter) string {
	clusterID := ClusterID(getter)
	return fmt.Sprintf("%s-master-0", clusterID)
}

func ToCluster(v interface{}) (capiv1alpha3.Cluster, error) {
	if v == nil {
		return capiv1alpha3.Cluster{}, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", &capiv1alpha3.Cluster{}, v)
	}

	customObjectPointer, ok := v.(*capiv1alpha3.Cluster)
	if !ok {
		return capiv1alpha3.Cluster{}, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", &capiv1alpha3.Cluster{}, v)
	}
	customObject := *customObjectPointer

	return customObject, nil
}

func ToAzureMachine(v interface{}) (capzv1alpha3.AzureMachine, error) {
	if v == nil {
		return capzv1alpha3.AzureMachine{}, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", &capzv1alpha3.AzureMachine{}, v)
	}

	customObjectPointer, ok := v.(*capzv1alpha3.AzureMachine)
	if !ok {
		return capzv1alpha3.AzureMachine{}, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", &capzv1alpha3.AzureMachine{}, v)
	}
	customObject := *customObjectPointer

	return customObject, nil
}

func ToAzureMachinePool(v interface{}) (expcapzv1alpha3.AzureMachinePool, error) {
	if v == nil {
		return expcapzv1alpha3.AzureMachinePool{}, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", &expcapzv1alpha3.AzureMachinePool{}, v)
	}

	customObjectPointer, ok := v.(*expcapzv1alpha3.AzureMachinePool)
	if !ok {
		return expcapzv1alpha3.AzureMachinePool{}, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", &expcapzv1alpha3.AzureMachinePool{}, v)
	}
	customObject := *customObjectPointer

	return customObject, nil
}

func ToMachinePool(v interface{}) (expcapiv1alpha3.MachinePool, error) {
	if v == nil {
		return expcapiv1alpha3.MachinePool{}, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", &expcapiv1alpha3.MachinePool{}, v)
	}

	customObjectPointer, ok := v.(*expcapiv1alpha3.MachinePool)
	if !ok {
		return expcapiv1alpha3.MachinePool{}, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", &expcapiv1alpha3.MachinePool{}, v)
	}
	customObject := *customObjectPointer

	return customObject, nil
}

func BlobContainerName() string {
	return blobContainerName
}

func BlobName(customObject LabelsGetter, role string) string {
	return fmt.Sprintf("%s-%s-%s", OperatorVersion(customObject), cloudConfigVersion, role)
}

func WorkerBlobName(operatorVersion string) string {
	return fmt.Sprintf("%s-%s-%s", operatorVersion, cloudConfigVersion, prefixWorker)
}

func BootstrapBlobName(customObject expcapzv1alpha3.AzureMachinePool) string {
	return fmt.Sprintf("%s-%s-%s", ClusterID(&customObject), customObject.Name, OperatorVersion(&customObject))
}

func CalicoCIDR(customObject providerv1alpha1.AzureConfig) string {
	return customObject.Spec.Azure.VirtualNetwork.CalicoSubnetCIDR
}

func CertificateEncryptionSecretName(customObject LabelsGetter) string {
	return fmt.Sprintf("%s-certificate-encryption", ClusterID(customObject))
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

func ClusterIPRange(customObject providerv1alpha1.AzureConfig) string {
	return customObject.Spec.Cluster.Kubernetes.API.ClusterIPRange
}

func ClusterName(getter LabelsGetter) string {
	return getter.GetLabels()[capiv1alpha3.ClusterLabelName]
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
		clusterTagName:      to.StringPtr(ClusterID(&customObject)),
		installationTagName: to.StringPtr(installationName),
		organizationTagName: to.StringPtr(ClusterOrganization(customObject)),
	}

	return tags
}

// CredentialName returns name of the credential secret.
func CredentialName(customObject providerv1alpha1.AzureConfig) string {
	name := customObject.Spec.Azure.CredentialSecret.Name
	if name == "" {
		name = credentialDefaultName
	}
	return name
}

// CredentialNamespace returns namespace of the credential secret.
func CredentialNamespace(customObject providerv1alpha1.AzureConfig) string {
	namespace := customObject.Spec.Azure.CredentialSecret.Namespace
	if namespace == "" {
		namespace = credentialDefaultNamespace
	}
	return namespace
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
	return fmt.Sprintf("%s.k8s", ClusterID(&customObject))
}

// DNSZonePrefixEtcd returns relative name of the etcd DNS zone.
func DNSZonePrefixEtcd(customObject providerv1alpha1.AzureConfig) string {
	return fmt.Sprintf("%s.k8s", ClusterID(&customObject))
}

// DNSZonePrefixIngress returns relative name of the ingress DNS zone.
func DNSZonePrefixIngress(customObject providerv1alpha1.AzureConfig) string {
	return fmt.Sprintf("%s.k8s", ClusterID(&customObject))
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

func InstanceIDFromNode(node v1.Node) (string, error) {
	// Extract "000001" from "nodepool-w8kjg-000001".
	var base36 string
	{
		base36 = instanceIDRegexp.FindString(node.Name)
		if base36 == "" {
			return "", microerror.Maskf(executionFailedError, "Unable to extract instance ID from node name")
		}
		base36 = strings.TrimPrefix(base36, "-")
	}

	i, err := strconv.ParseUint(base36, 36, 64)
	if err != nil {
		return "", microerror.Maskf(executionFailedError, "expected VMSS instanceID to be a positive integer number but got %#q", i)
	}

	return strconv.FormatUint(i, 10), nil
}

// IsClusterCreating check if the cluster is being created.
func IsClusterCreating(cr providerv1alpha1.AzureConfig) bool {
	// When cluster creation is in the beginning, it doesn't necessarily have
	// any status conditions yet.
	if len(cr.Status.Cluster.Conditions) == 0 {
		return true
	}
	if cr.Status.Cluster.HasCreatingCondition() {
		return true
	}

	return false
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

// These are the same labels that kubernetesd adds when creating/updating an AzureConfig.
func KubeletLabelsNodePool(getter LabelsGetter) (string, error) {
	var labels string

	labels = ensureLabel(labels, label.Provider, "azure")
	labels = ensureLabel(labels, label.OperatorVersion, OperatorVersion(getter))
	labels = ensureLabel(labels, label.ReleaseVersion, ReleaseVersion(getter))

	machinePoolID, err := MachinePoolID(getter)
	if err != nil || machinePoolID == "" {
		return labels, microerror.Mask(missingMachinePoolLabelError)
	}

	labels = ensureLabel(labels, apiextensionslabels.MachinePool, machinePoolID)

	return labels, nil
}

// MasterSecurityGroupName returns name of the security group attached to master subnet.
func MasterSecurityGroupName(customObject providerv1alpha1.AzureConfig) string {
	return fmt.Sprintf("%s-%s", ClusterID(&customObject), masterSecurityGroupSuffix)
}

// WorkerCount returns the desired number of workers. Kept for the sake of compiling old instance
// resource. It should be removed as soon as instance resource is removed.
func WorkerCount(customObject providerv1alpha1.AzureConfig) int {
	return len(customObject.Spec.Azure.Workers)
}

// WorkerSecurityGroupName returns name of the security group attached to worker subnet.
func WorkerSecurityGroupName(customObject providerv1alpha1.AzureConfig) string {
	return fmt.Sprintf("%s-%s", ClusterID(&customObject), workerSecurityGroupSuffix)
}

// MasterSubnetName returns name of the master subnet.
func MasterSubnetName(customObject providerv1alpha1.AzureConfig) string {
	return fmt.Sprintf("%s-%s-%s", ClusterID(&customObject), virtualNetworkSuffix, masterSubnetSuffix)
}

func MasterSubnetNameFromClusterAPIObject(getter LabelsGetter) string {
	return fmt.Sprintf("%s-%s-%s", ClusterName(getter), virtualNetworkSuffix, masterSubnetSuffix)
}

func MastersSubnetCIDR(customObject providerv1alpha1.AzureConfig) string {
	return customObject.Spec.Azure.VirtualNetwork.MasterSubnetCIDR
}

// WorkerSubnetName returns name of the worker subnet.
func WorkerSubnetName(customObject providerv1alpha1.AzureConfig) string {
	return fmt.Sprintf("%s-%s-%s", ClusterID(&customObject), virtualNetworkSuffix, workerSubnetSuffix)
}

func MasterInstanceName(customObject providerv1alpha1.AzureConfig, instanceID string) string {
	idB36, err := vmssInstanceIDBase36(instanceID)
	if err != nil {
		panic(err)
	}

	return fmt.Sprintf("%s-master-%s-%06s", ClusterID(&customObject), ClusterID(&customObject), idB36)
}

func MasterNatGatewayID(cr providerv1alpha1.AzureConfig, subscriptionID string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/natGateways/%s", subscriptionID, ResourceGroupName(cr), masterNatGatewayName)
}

// MasterNICName returns name of the master NIC.
func MasterNICName(customObject providerv1alpha1.AzureConfig) string {
	return fmt.Sprintf("%s-master-%s-nic", ClusterID(&customObject), ClusterID(&customObject))
}

func MasterVMSSName(customObject providerv1alpha1.AzureConfig) string {
	return fmt.Sprintf("%s-master-%s", ClusterID(&customObject), ClusterID(&customObject))
}

func MasterVMSSNameFromClusterAPIObject(getter LabelsGetter) string {
	return fmt.Sprintf("%s-master-%s", ClusterName(getter), ClusterName(getter))
}

func PrefixMaster() string {
	return prefixMaster
}

func PrefixWorker() string {
	return prefixWorker
}

// ResourceGroupName returns name of the resource group for this cluster.
func ResourceGroupName(customObject providerv1alpha1.AzureConfig) string {
	return ClusterID(&customObject)
}

// RouteTableName returns name of the route table for this cluster.
func RouteTableName(customObject providerv1alpha1.AzureConfig) string {
	return fmt.Sprintf("%s-%s", ClusterID(&customObject), routeTableSuffix)
}

// AvailabilityZones returns the availability zones where the cluster will be created.
func AvailabilityZones(customObject providerv1alpha1.AzureConfig, location string) []int {
	if customObject.Spec.Azure.AvailabilityZones == nil {
		return []int{}
	}

	return customObject.Spec.Azure.AvailabilityZones
}

func ScalingMinWorkers(customObject providerv1alpha1.AzureConfig) int {
	return customObject.Spec.Cluster.Scaling.Min
}

func ScalingMaxWorkers(customObject providerv1alpha1.AzureConfig) int {
	return customObject.Spec.Cluster.Scaling.Max
}

func StorageAccountName(customObject LabelsGetter) string {
	// In integration tests we use hyphens which are not allowed. We also
	// need to keep the name globaly unique and within 24 character limit.
	//
	//	See https://docs.microsoft.com/en-us/azure/architecture/best-practices/naming-conventions#storage
	//
	storageAccountName := fmt.Sprintf("%s%s", storageAccountSuffix, ClusterID(customObject))
	return strings.Replace(storageAccountName, "-", "", -1)
}

func ToAzureCluster(v interface{}) (capzv1alpha3.AzureCluster, error) {
	if v == nil {
		return capzv1alpha3.AzureCluster{}, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", &capzv1alpha3.AzureCluster{}, v)
	}

	customObjectPointer, ok := v.(*capzv1alpha3.AzureCluster)
	if !ok {
		return capzv1alpha3.AzureCluster{}, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", &capzv1alpha3.AzureCluster{}, v)
	}
	customObject := *customObjectPointer

	return customObject, nil
}

func ToClusterEndpoint(v interface{}) (string, error) {
	cr, err := ToCustomResource(v)
	if err != nil {
		return "", microerror.Mask(err)
	}

	return cr.Spec.Cluster.Kubernetes.API.Domain, nil
}

func ToClusterKubernetesSSHUser(s employees.SSHUserList) []v1alpha1.ClusterKubernetesSSHUser {
	var ret []v1alpha1.ClusterKubernetesSSHUser

	for name, keys := range s {
		ret = append(ret, v1alpha1.ClusterKubernetesSSHUser{
			Name: name,
			// v1alpha1 type currently supports only one ssh key per user.
			PublicKey: keys[0],
		})
	}

	return ret
}

func ToClusterID(v interface{}) (string, error) {
	cr, err := ToCustomResource(v)
	if err != nil {
		return "", microerror.Mask(err)
	}

	return ClusterID(&cr), nil
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

	return OperatorVersion(&customObject), nil
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

// VnetName returns name of the virtual network.
func VnetName(customObject providerv1alpha1.AzureConfig) string {
	return fmt.Sprintf("%s-%s", ClusterID(&customObject), virtualNetworkSuffix)
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
	return fmt.Sprintf("%s-%s", ClusterID(&customObject), vpnGatewaySuffix)
}

func VPNGatewayPublicIPName(customObject providerv1alpha1.AzureConfig) string {
	return fmt.Sprintf("%s-VPNGateway-PublicIP", ClusterID(&customObject))
}

func WorkerInstanceName(clusterID, instanceID string) string {
	idB36, err := vmssInstanceIDBase36(instanceID)
	if err != nil {
		panic(err)
	}

	return fmt.Sprintf("%s-worker-%s-%06s", clusterID, clusterID, idB36)
}

func NodePoolDeploymentName(azureMachinePool *expcapzv1alpha3.AzureMachinePool) string {
	return NodePoolVMSSName(azureMachinePool)
}

func SubnetDeploymentName(subnetName string) string {
	return fmt.Sprintf("%s-%s", subnetDeploymentPrefix, subnetName)
}

func MachinePoolID(getter LabelsGetter) (string, error) {
	machinePoolID, exists := getter.GetLabels()[apiextensionslabels.MachinePool]
	if !exists {
		return "", microerror.Mask(missingMachinePoolLabelError)
	}

	return machinePoolID, nil
}

func NodePoolInstanceName(nodePoolName, instanceID string) string {
	idB36, err := vmssInstanceIDBase36(instanceID)
	if err != nil {
		panic(err)
	}

	return fmt.Sprintf("nodepool-%s-%06s", nodePoolName, idB36)
}

func NodePoolMinReplicas(machinePool *expcapiv1alpha3.MachinePool) int32 {
	sizeStr := machinePool.Annotations[apiextensionsannotations.NodePoolMinSize]
	size, err := strconv.Atoi(sizeStr) // nolint:gosec
	if err != nil {
		// Annotation not found or invalid.
		return *machinePool.Spec.Replicas
	}

	return int32(size)
}

func NodePoolMaxReplicas(machinePool *expcapiv1alpha3.MachinePool) int32 {
	sizeStr := machinePool.Annotations[apiextensionsannotations.NodePoolMaxSize]
	size, err := strconv.Atoi(sizeStr) // nolint:gosec
	if err != nil {
		// Annotation not found or invalid.
		return *machinePool.Spec.Replicas
	}

	return int32(size)
}

func NodePoolVMSSName(azureMachinePool *expcapzv1alpha3.AzureMachinePool) string {
	return fmt.Sprintf("%s-%s", "nodepool", azureMachinePool.Name)
}

func WorkerVMSSName(customObject providerv1alpha1.AzureConfig) string {
	return fmt.Sprintf("%s-worker-%s", ClusterID(&customObject), ClusterID(&customObject))
}

func WorkersSubnetCIDR(customObject providerv1alpha1.AzureConfig) string {
	return customObject.Spec.Azure.VirtualNetwork.WorkerSubnetCIDR
}

func NodePoolSpotInstancesEnabled(azureMachinePool *expcapzv1alpha3.AzureMachinePool) bool {
	return azureMachinePool.Spec.Template.SpotVMOptions != nil
}

func NodePoolSpotInstancesMaxPrice(azureMachinePool *expcapzv1alpha3.AzureMachinePool) string {
	if azureMachinePool.Spec.Template.SpotVMOptions == nil || azureMachinePool.Spec.Template.SpotVMOptions.MaxPrice == nil {
		return ""
	}

	return azureMachinePool.Spec.Template.SpotVMOptions.MaxPrice.AsDec().String()
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

func ensureLabel(labels string, key string, value string) string {
	if key == "" {
		return labels
	}
	if value == "" {
		return labels
	}

	var split []string
	if labels != "" {
		split = strings.Split(labels, ",")
	}

	var found bool
	for i, l := range split {
		if !strings.HasPrefix(l, key+"=") {
			continue
		}

		found = true
		split[i] = key + "=" + value
	}

	if !found {
		split = append(split, key+"="+value)
	}

	joined := strings.Join(split, ",")

	return joined
}
