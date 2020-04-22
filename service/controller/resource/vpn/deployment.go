package vpn

import (
	"encoding/json"

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

	template, err := getARMTemplate()
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
func getARMTemplate() (*map[string]interface{}, error) {
	contents := make(map[string]interface{})
	const data string = `{
  "$schema":"https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
  "contentVersion":"1.0.0.0",
  "parameters":{
    "clusterID":{
      "type":"string"
    },
    "GiantSwarmTags":{
      "type":"object",
      "defaultValue":{
        "provider":"F80D01C0-7AAC-4440-98F6-5061511962AD"
      }
    },
    "virtualNetworkName":{
      "type":"string"
    },
    "vnetGatewaySubnetName":{
      "type":"string"
    },
    "vpnGatewayName":{
      "type":"string"
    }
  },
  "variables":{
    "publicIPName":"[concat(parameters('clusterID'), '-VPNGateway-PublicIP')]",
    "publicIPID":"[resourceId('Microsoft.Network/publicIPAddresses', variables('publicIPName'))]",
    "vnetID":"[resourceId('Microsoft.Network/virtualNetworks/subnets', parameters('virtualNetworkName'), parameters('vnetGatewaySubnetName'))]",
    "publicIPAddressesAPIVersion":"2017-10-01",
    "virtualNetworksGatewayAPIVersion":"2017-10-01"
  },
  "resources":[
    {
      "name":"[variables('publicIPName')]",
      "type":"Microsoft.Network/publicIPAddresses",
      "apiVersion":"[variables('publicIPAddressesAPIVersion')]",
      "location":"[resourceGroup().location]",
      "tags":{
        "provider":"[toUpper(parameters('GiantSwarmTags').provider)]"
      },
      "sku":{
        "name": "Basic"
      },
      "properties":{
        "publicIPAllocationMethod":"Dynamic"
      }
    },
    {
      "type":"Microsoft.Network/virtualNetworkGateways",
      "name":"[parameters('vpnGatewayName')]",
      "apiVersion":"[variables('virtualNetworksGatewayAPIVersion')]",
      "location":"[resourceGroup().location]",
      "tags":{
        "provider":"[toUpper(parameters('GiantSwarmTags').provider)]"
      },
      "dependsOn":[
        "[concat('Microsoft.Network/publicIPAddresses/', variables('publicIPName'))]"
      ],
      "properties":{
        "ipConfigurations":[
          {
            "id":"string",
            "properties":{
              "privateIPAllocationMethod":"Dynamic",
              "subnet":{
                "id":"[variables('vnetID')]"
              },
              "publicIPAddress":{
                "id":"[variables('publicIPID')]"
              }
            },
            "name":"string"
          }
        ],
        "gatewayType":"Vpn",
        "vpnType":"RouteBased",
        "vpnClientConfiguration":{
          "vpnClientProtocols":[
            "SSTP",
            "IkeV2"
          ]
        },
        "sku":{
          "name":"VpnGw1",
          "tier":"VpnGw1"
        }
      }
    }
  ]
}
`
	if err := json.Unmarshal([]byte(data), &contents); err != nil {
		return nil, err
	}
	return &contents, nil
}
