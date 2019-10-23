package vpn

import (
	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest/to"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"

	"github.com/giantswarm/azure-operator/service/controller/v10patch2/key"
)

func (r Resource) newDeployment(customObject providerv1alpha1.AzureConfig, overwrites map[string]interface{}) azureresource.Deployment {
	defaultParams := map[string]interface{}{
		"clusterID":             key.ClusterID(customObject),
		"virtualNetworkName":    key.VnetName(customObject),
		"vnetGatewaySubnetName": key.VNetGatewaySubnetName(),
		"vpnGatewayName":        key.VPNGatewayName(customObject),
	}

	d := azureresource.Deployment{
		Properties: &azureresource.DeploymentProperties{
			Mode:       azureresource.Incremental,
			Parameters: key.ToParameters(defaultParams, overwrites),
			TemplateLink: &azureresource.TemplateLink{
				URI:            to.StringPtr(key.ARMTemplateURI(r.templateVersion, "vpn", "main.json")),
				ContentVersion: to.StringPtr(key.TemplateContentVersion),
			},
		},
	}

	return d
}
