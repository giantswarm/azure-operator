package deployment

import (
	"context"

	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/Azure/go-autorest/autorest/to"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/service/controller/controllercontext"
	"github.com/giantswarm/azure-operator/service/controller/key"
)

func (r Resource) newDeployment(ctx context.Context, customObject providerv1alpha1.AzureConfig, overwrites map[string]interface{}) (azureresource.Deployment, error) {
	cc, err := controllercontext.FromContext(ctx)
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
		"storageAccountName":      key.StorageAccountName(customObject),
		"virtualNetworkCidr":      key.VnetCIDR(customObject),
		"virtualNetworkName":      key.VnetName(customObject),
		"vnetGatewaySubnetName":   key.VNetGatewaySubnetName(),
		"vpnSubnetCidr":           cc.AzureNetwork.VPN.String(),
		"workerSubnetCidr":        cc.AzureNetwork.Worker.String(),
	}

	d := azureresource.Deployment{
		Properties: &azureresource.DeploymentProperties{
			Mode:       azureresource.Incremental,
			Parameters: key.ToParameters(defaultParams, overwrites),
			TemplateLink: &azureresource.TemplateLink{
				URI:            to.StringPtr(key.ARMTemplateURI(r.templateVersion, "deployment", "main.json")),
				ContentVersion: to.StringPtr(key.TemplateContentVersion),
			},
		},
	}

	return d, nil
}
