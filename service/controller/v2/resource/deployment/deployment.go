package deployment

import (
	"context"

	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-02-01/resources"
	"github.com/Azure/go-autorest/autorest/to"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"

	"github.com/giantswarm/azure-operator/service/controller/v2/key"
)

func (r Resource) newDeployment(ctx context.Context, obj providerv1alpha1.AzureConfig, overwrites map[string]interface{}) (azureresource.Deployment, error) {
	defaultParams := map[string]interface{}{
		"calicoSubnetCidr":              key.VnetCalicoSubnetCIDR(obj),
		"clusterID":                     key.ClusterID(obj),
		"dnsZones":                      obj.Spec.Azure.DNSZones,
		"hostClusterCidr":               r.azure.HostCluster.CIDR,
		"hostClusterResourceGroupName":  r.azure.HostCluster.ResourceGroup,
		"hostClusterVirtualNetworkName": r.azure.HostCluster.VirtualNetwork,
		"kubernetesAPISecurePort":       obj.Spec.Cluster.Kubernetes.API.SecurePort,
		"masterSubnetCidr":              key.VnetMasterSubnetCIDR(obj),
		"templatesBaseURI":              baseTemplateURI(r.templateVersion),
		"virtualNetworkCidr":            key.VnetCIDR(obj),
		"virtualNetworkName":            key.VnetName(obj),
		"workerSubnetCidr":              key.VnetWorkerSubnetCIDR(obj),
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
