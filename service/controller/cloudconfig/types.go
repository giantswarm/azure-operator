package cloudconfig

import (
	providerv1alpha1 "github.com/giantswarm/apiextensions/v2/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/certs/v3/pkg/certs"
	k8scloudconfig "github.com/giantswarm/k8scloudconfig/v8/pkg/template"
	capzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	expcapzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	expcapiv1alpha3 "sigs.k8s.io/cluster-api/exp/api/v1alpha3"
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
	AzureMachinePool *expcapzv1alpha3.AzureMachinePool
	AzureCluster     *capzv1alpha3.AzureCluster
	Cluster          *capiv1alpha3.Cluster
	CustomObject     providerv1alpha1.AzureConfig
	Images           k8scloudconfig.Images
	MachinePool      *expcapiv1alpha3.MachinePool
	MasterCertFiles  []certs.File
	Versions         k8scloudconfig.Versions
	WorkerCertFiles  []certs.File
}
