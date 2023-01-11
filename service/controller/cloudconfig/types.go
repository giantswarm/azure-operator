package cloudconfig

import (
	providerv1alpha1 "github.com/giantswarm/apiextensions/v6/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/certs/v4/pkg/certs"
	k8scloudconfig "github.com/giantswarm/k8scloudconfig/v15/pkg/template"
	capz "sigs.k8s.io/cluster-api-provider-azure/api/v1beta1"
	capzexp "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1beta1"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
	capiexp "sigs.k8s.io/cluster-api/exp/api/v1beta1"
)

type templateData struct {
	azureCNIFileParams
	calicoAzureFileParams
	cloudProviderConfFileParams
	certificateDecrypterUnitParams
	ingressLBFileParams
}

type azureCNIFileParams struct {
	VnetCIDR string
}

type calicoAzureFileParams struct {
	Cluster    providerv1alpha1.Cluster
	CalicoCIDR string
}

type cloudProviderConfFileParams struct {
	AADClientID                 string
	AADClientSecret             string
	EnvironmentName             string
	Location                    string
	PrimaryScaleSetName         string
	ResourceGroup               string
	RouteTableName              string
	SecurityGroupName           string
	SubnetName                  string
	SubscriptionID              string
	TenantID                    string
	VnetName                    string
	UseManagedIdentityExtension bool
}

type certificateDecrypterUnitParams struct {
	CertsPaths []string
}

type ingressLBFileParams struct {
	ClusterDNSDomain string
}

type IgnitionTemplateData struct {
	AzureMachinePool *capzexp.AzureMachinePool
	AzureCluster     *capz.AzureCluster
	Cluster          *capi.Cluster
	CustomObject     providerv1alpha1.AzureConfig
	EncryptionConf   []byte
	Images           k8scloudconfig.Images
	MachinePool      *capiexp.MachinePool
	MasterCertFiles  []certs.File
	Versions         k8scloudconfig.Versions
	WorkerCertFiles  []certs.File
}
