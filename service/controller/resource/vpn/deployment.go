package vpn

import (
	"encoding/json"
	"io/ioutil"

	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/service/controller/key"
)

func (r Resource) newDeployment(customObject providerv1alpha1.AzureConfig, overwrites map[string]interface{}) (azureresource.Deployment, error) {
	defaultParams := map[string]interface{}{
		"clusterID":             key.ClusterID(customObject),
		"virtualNetworkName":    key.VnetName(customObject),
		"vnetGatewaySubnetName": key.VNetGatewaySubnetName(),
		"vpnGatewayName":        key.VPNGatewayName(customObject),
	}

	template, err := getARMTemplate("service/controller/resource/vpn/template/main.json")
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	d := azureresource.Deployment{
		Properties: &azureresource.DeploymentProperties{
			Mode:       azureresource.Incremental,
			Parameters: key.ToParameters(defaultParams, overwrites),
			Template:   template,
		},
	}

	return d, nil
}

// getARMTemplate reads a json file, and unmarshals it.
func getARMTemplate(path string) (*map[string]interface{}, error) {
	contents := make(map[string]interface{})
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &contents); err != nil {
		return nil, err
	}
	return &contents, nil
}
