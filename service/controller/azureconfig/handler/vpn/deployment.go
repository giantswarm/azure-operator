package vpn

import (
	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	providerv1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v5/service/controller/azureconfig/handler/vpn/template"
	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

func (r Resource) newDeployment(customObject providerv1alpha1.AzureConfig, overwrites map[string]interface{}) (azureresource.Deployment, error) {
	defaultParams := map[string]interface{}{
		"clusterID":             key.ClusterID(&customObject),
		"virtualNetworkName":    key.VnetName(customObject),
		"vnetGatewaySubnetName": key.VNetGatewaySubnetName(),
		"vpnGatewayName":        key.VPNGatewayName(customObject),
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
