package deployment

import (
	"context"

	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v3/service/controller/controllercontext"
	"github.com/giantswarm/azure-operator/v3/service/controller/key"
	"github.com/giantswarm/azure-operator/v3/service/controller/resource/deployment/template"
)

func (r Resource) newDeployment(ctx context.Context, customObject providerv1alpha1.AzureConfig, overwrites map[string]interface{}) (azureresource.Deployment, error) {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	groupsClient, err := r.getGroupsClient(ctx)
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	group, err := groupsClient.Get(ctx, key.ClusterID(customObject))
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	defaultParams := map[string]interface{}{
		"blobContainerName":       key.BlobContainerName(),
		"calicoSubnetCidr":        cc.AzureNetwork.Calico.String(),
		"clusterID":               key.ClusterID(customObject),
		"dnsZones":                key.DNSZones(customObject),
		"hostClusterCidr":         r.azure.HostCluster.CIDR,
		"kubernetesAPISecurePort": key.APISecurePort(customObject),
		"masterSubnetCidr":        cc.AzureNetwork.Master.String(),
		"natGwZones":              key.AvailabilityZones(customObject, *group.Location),
		"storageAccountName":      key.StorageAccountName(customObject),
		"virtualNetworkCidr":      key.VnetCIDR(customObject),
		"virtualNetworkName":      key.VnetName(customObject),
		"vnetGatewaySubnetName":   key.VNetGatewaySubnetName(),
		"vpnSubnetCidr":           cc.AzureNetwork.VPN.String(),
		"workerSubnetCidr":        cc.AzureNetwork.Worker.String(),
	}

	armTemplate, err := template.GetARMTemplate()
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	d := azureresource.Deployment{
		Properties: &azureresource.DeploymentProperties{
			Mode:       azureresource.Incremental,
			Parameters: key.ToParameters(defaultParams, overwrites),
			Template:   armTemplate,
		},
	}

	return d, nil
}
