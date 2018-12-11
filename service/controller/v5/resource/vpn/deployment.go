package vpn

//go:generate go run github.com/logrusorgru/textFileToGoConst -in virtual_network_gateway.json -c vpn_gateway_json

import (
	"encoding/json"

	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/service/controller/v5/key"
)

func (r Resource) newDeployment(template string, customObject providerv1alpha1.AzureConfig, overwrites map[string]interface{}) (azureresource.Deployment, error) {
	defaultParams := map[string]interface{}{
		"clusterID":             key.ClusterID(customObject),
		"virtualNetworkName":    key.VnetName(customObject),
		"vnetGatewaySubnetName": key.VNetGatewaySubnetName(),
		"vpnGatewayName":        key.VPNGatewayName(customObject),
	}

	var tmpl map[string]interface{}
	err := json.Unmarshal([]byte(template), &tmpl)
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	d := azureresource.Deployment{
		Properties: &azureresource.DeploymentProperties{
			Mode:       azureresource.Incremental,
			Parameters: key.ToParameters(defaultParams, overwrites),
			Template:   tmpl,
		},
	}

	return d, nil
}
