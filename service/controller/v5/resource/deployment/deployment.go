package deployment

import (
	"context"

	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest/to"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/service/controller/v5/controllercontext"
	"github.com/giantswarm/azure-operator/service/controller/v5/key"
)

const (
	mainTemplate = "main.json"
)

func (r Resource) newDeployment(ctx context.Context, customObject providerv1alpha1.AzureConfig, overwrites map[string]interface{}) (azureresource.Deployment, error) {
	sc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	defaultParams := map[string]interface{}{
		"blobContainerName":       key.BlobContainerName(),
		"calicoSubnetCidr":        sc.AzureNetwork.Calico.String(),
		"clusterID":               key.ClusterID(customObject),
		"dnsZones":                key.DNSZones(customObject),
		"hostClusterCidr":         r.azure.HostCluster.CIDR,
		"kubernetesAPISecurePort": key.APISecurePort(customObject),
		"masterSubnetCidr":        sc.AzureNetwork.Master.String(),
		"storageAccountName":      key.StorageAccountName(customObject),
		"templatesBaseURI":        key.TemplateBaseURI(r.templateVersion, "deployment"),
		"virtualNetworkCidr":      key.VnetCIDR(customObject),
		"virtualNetworkName":      key.VnetName(customObject),
		"vpnGatewayName":          key.VPNGatewayName(customObject),
		"vpnSubnetCidr":           sc.AzureNetwork.VPN.String(),
		"workerSubnetCidr":        sc.AzureNetwork.Worker.String(),
	}

	d := azureresource.Deployment{
		Properties: &azureresource.DeploymentProperties{
			Mode:       azureresource.Incremental,
			Parameters: key.ToParameters(defaultParams, overwrites),
			TemplateLink: &azureresource.TemplateLink{
				URI:            to.StringPtr(key.TemplateURI(r.templateVersion, "deployment", mainTemplate)),
				ContentVersion: to.StringPtr(key.TemplateContentVersion),
			},
		},
	}

	return d, nil
}
