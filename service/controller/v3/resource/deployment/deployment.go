package deployment

import (
	"context"

	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-02-01/resources"
	"github.com/Azure/go-autorest/autorest/to"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"

	"github.com/giantswarm/azure-operator/service/controller/v3/controllercontext"
	"github.com/giantswarm/azure-operator/service/controller/v3/key"
)

func (r Resource) newDeployment(ctx context.Context, customObject providerv1alpha1.AzureConfig, overwrites map[string]interface{}) (azureresource.Deployment, error) {
	defaultParams := map[string]interface{}{
		"calicoSubnetCidr":              key.VnetCalicoSubnetCIDR(customObject),
		"clusterID":                     key.ClusterID(customObject),
		"dnsZones":                      key.DNSZones(customObject),
		"hostClusterCidr":               r.azure.HostCluster.CIDR,
		"hostClusterResourceGroupName":  r.azure.HostCluster.ResourceGroup,
		"hostClusterVirtualNetworkName": r.azure.HostCluster.VirtualNetwork,
		"kubernetesAPISecurePort":       key.APISecurePort(customObject),
		"masterSubnetCidr":              key.VnetMasterSubnetCIDR(customObject),
		"templatesBaseURI":              baseTemplateURI(r.templateVersion),
		"virtualNetworkCidr":            key.VnetCIDR(customObject),
		"virtualNetworkName":            key.VnetName(customObject),
		"workerSubnetCidr":              key.VnetWorkerSubnetCIDR(customObject),
	}

	d := azureresource.Deployment{
		Properties: &azureresource.DeploymentProperties{
			Mode:       azureresource.Incremental,
			Parameters: key.ToParameters(defaultParams, overwrites),
			TemplateLink: &azureresource.TemplateLink{
				URI:            to.StringPtr(templateURI(r.templateVersion, mainTemplate)),
				ContentVersion: to.StringPtr(key.TemplateContentVersion),
			},
		},
	}

	return d, nil
}
