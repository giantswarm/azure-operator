package v1alpha1

import (
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/apiextensions/pkg/crd"
)

const (
	kindAzureConfig = "AzureConfig"
)

func NewAzureConfigCRD() *v1beta1.CustomResourceDefinition {
	return crd.LoadV1Beta1(group, kindAzureConfig)
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status

type AzureConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              AzureConfigSpec `json:"spec"`
	// +kubebuilder:validation:Optional
	Status AzureConfigStatus `json:"status" yaml:"status"`
}

type AzureConfigSpec struct {
	Cluster       Cluster                      `json:"cluster" yaml:"cluster"`
	Azure         AzureConfigSpecAzure         `json:"azure" yaml:"azure"`
	VersionBundle AzureConfigSpecVersionBundle `json:"versionBundle" yaml:"versionBundle"`
}

type AzureConfigSpecAzure struct {
	AvailabilityZones []int                              `json:"availabilityZones" yaml:"availabilityZones"`
	CredentialSecret  CredentialSecret                   `json:"credentialSecret" yaml:"credentialSecret"`
	DNSZones          AzureConfigSpecAzureDNSZones       `json:"dnsZones" yaml:"dnsZones"`
	Masters           []AzureConfigSpecAzureNode         `json:"masters" yaml:"masters"`
	VirtualNetwork    AzureConfigSpecAzureVirtualNetwork `json:"virtualNetwork" yaml:"virtualNetwork"`
	Workers           []AzureConfigSpecAzureNode         `json:"workers" yaml:"workers"`
}

// AzureConfigSpecAzureDNSZones contains the DNS Zones of the cluster.
type AzureConfigSpecAzureDNSZones struct {
	// API is the DNS Zone for the Kubernetes API.
	API AzureConfigSpecAzureDNSZonesDNSZone `json:"api" yaml:"api"`
	// Etcd is the DNS Zone for the etcd cluster.
	Etcd AzureConfigSpecAzureDNSZonesDNSZone `json:"etcd" yaml:"etcd"`
	// Ingress is the DNS Zone for the Ingress resource, used for customer traffic.
	Ingress AzureConfigSpecAzureDNSZonesDNSZone `json:"ingress" yaml:"ingress"`
}

// AzureConfigSpecAzureDNSZonesDNSZone points to a DNS Zone in Azure.
type AzureConfigSpecAzureDNSZonesDNSZone struct {
	// ResourceGroup is the resource group of the zone.
	ResourceGroup string `json:"resourceGroup" yaml:"resourceGroup"`
	// Name is the name of the zone.
	Name string `json:"name" yaml:"name"`
}

type AzureConfigSpecAzureVirtualNetwork struct {
	// CIDR is the CIDR for the Virtual Network.
	CIDR string `json:"cidr" yaml:"cidr"`

	// TODO: remove Master, Worker and Calico subnet cidr after azure-operator v2
	// is deleted. MasterSubnetCIDR is the CIDR for the master subnet.
	//
	//     https://github.com/giantswarm/giantswarm/issues/4358
	//
	MasterSubnetCIDR string `json:"masterSubnetCIDR" yaml:"masterSubnetCIDR"`
	// WorkerSubnetCIDR is the CIDR for the worker subnet.
	WorkerSubnetCIDR string `json:"workerSubnetCIDR" yaml:"workerSubnetCIDR"`

	// CalicoSubnetCIDR is the CIDR for the calico subnet. It has to be
	// also a worker subnet (Azure limitation).
	CalicoSubnetCIDR string `json:"calicoSubnetCIDR" yaml:"calicoSubnetCIDR"`
}

type AzureConfigSpecAzureNode struct {
	// VMSize is the master vm size (e.g. Standard_A1)
	VMSize string `json:"vmSize" yaml:"vmSize"`
	// DockerVolumeSizeGB is the size of a volume mounted to /var/lib/docker.
	DockerVolumeSizeGB int `json:"dockerVolumeSizeGB" yaml:"dockerVolumeSizeGB"`
	// KubeletVolumeSizeGB is the size of a volume mounted to /var/lib/kubelet.
	KubeletVolumeSizeGB int `json:"kubeletVolumeSizeGB" yaml:"kubeletVolumeSizeGB"`
}

type AzureConfigSpecVersionBundle struct {
	Version string `json:"version" yaml:"version"`
}

type AzureConfigStatus struct {
	// +kubebuilder:validation:Optional
	Cluster StatusCluster `json:"cluster" yaml:"cluster"`
	// +kubebuilder:validation:Optional
	Provider AzureConfigStatusProvider `json:"provider" yaml:"provider"`
}

type AzureConfigStatusProvider struct {
	// +kubebuilder:validation:Optional
	// +nullable
	AvailabilityZones []int `json:"availabilityZones,omitempty" yaml:"availabilityZones,omitempty"`
	// +kubebuilder:validation:Optional
	// +nullable
	Ingress AzureConfigStatusProviderIngress `json:"ingress" yaml:"ingress"`
}

type AzureConfigStatusProviderIngress struct {
	// +kubebuilder:validation:Optional
	// +nullable
	LoadBalancer AzureConfigStatusProviderIngressLoadBalancer `json:"loadBalancer" yaml:"loadBalancer"`
}

type AzureConfigStatusProviderIngressLoadBalancer struct {
	// +kubebuilder:validation:Optional
	PublicIPName string `json:"publicIPName" yaml:"publicIPName"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type AzureConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []AzureConfig `json:"items"`
}
