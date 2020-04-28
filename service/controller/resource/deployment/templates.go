package deployment

import (
	"encoding/json"
	"fmt"
)

const securityGroups string = `
{
  "$schema":"https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
  "contentVersion":"1.0.0.0",
  "parameters":{
    "networkSecurityGroupsAPIVersion":{
      "type":"string",
      "defaultValue":"2016-09-01",
      "metadata":{
        "description":"API version used by the Microsoft.Network/networkSecurityGroups resource."
      }
    },
    "clusterID":{
      "type":"string"
    },
    "GiantSwarmTags":{
      "type":"object",
      "defaultValue":{
        "provider":"F80D01C0-7AAC-4440-98F6-5061511962AD"
      }
    },
    "initialProvisioning":{
      "type":"string",
      "defaultValue":"Yes",
      "metadata":{
        "description":"Whether the deployment is provisioned the very first time."
      }
    },
    "virtualNetworkCidr":{
      "type":"string"
    },
    "calicoSubnetCidr":{
      "type":"string"
    },
    "masterSubnetCidr":{
      "type":"string"
    },
    "workerSubnetCidr":{
      "type":"string"
    },
    "hostClusterCidr":{
      "type":"string"
    },
    "kubernetesAPISecurePort":{
      "type":"int"
    }
  },
  "variables":{
    "masterSecurityGroupName":"[concat(parameters('clusterID'), '-MasterSecurityGroup')]",
    "workerSecurityGroupName":"[concat(parameters('clusterID'), '-WorkerSecurityGroup')]",
    "cadvisorPort":"4194",
    "etcdPort":"2379",
    "kubeletPort":"10250",
    "nodeExporterPort":"10300",
    "kubeStateMetricsPort":"10301"
  },
  "resources":[
    {
      "type":"Microsoft.Network/networkSecurityGroups",
      "name":"[variables('masterSecurityGroupName')]",
      "apiVersion":"[parameters('networkSecurityGroupsAPIVersion')]",
      "location":"[resourceGroup().location]",
      "tags":{
        "provider":"[toUpper(parameters('GiantSwarmTags').provider)]"
      },
      "properties":{
        "securityRules":[
          {
            "name":"defaultInboundRule",
            "properties":{
              "description":"Default rule that denies any inbound traffic.",
              "protocol":"*",
              "sourcePortRange":"*",
              "destinationPortRange":"*",
              "sourceAddressPrefix":"*",
              "destinationAddressPrefix":"*",
              "access":"Deny",
              "direction":"Inbound",
              "priority":"4096"
            }
          },
          {
            "name":"defaultOutboundRule",
            "properties":{
              "description":"Default rule that allows any outbound traffic.",
              "protocol":"*",
              "sourcePortRange":"*",
              "destinationPortRange":"*",
              "sourceAddressPrefix":"*",
              "destinationAddressPrefix":"*",
              "access":"Allow",
              "direction":"Outbound",
              "priority":"4095"
            }
          },
          {
            "name":"defaultInClusterRule",
            "properties":{
              "description":"Default rule that allows any traffic within the master subnet.",
              "protocol":"*",
              "sourcePortRange":"*",
              "destinationPortRange":"*",
              "sourceAddressPrefix":"[parameters('masterSubnetCidr')]",
              "destinationAddressPrefix":"[parameters('virtualNetworkCidr')]",
              "access":"Allow",
              "direction":"Inbound",
              "priority":"4094"
            }
          },
          {
            "name":"sshHostClusterToMasterSubnetRule",
            "properties":{
              "description":"Allow the host cluster machines to reach this guest cluster master subnet.",
              "protocol":"*",
              "sourcePortRange":"*",
              "destinationPortRange":"22",
              "sourceAddressPrefix":"[parameters('hostClusterCidr')]",
              "destinationAddressPrefix":"[parameters('masterSubnetCidr')]",
              "access":"Allow",
              "direction":"Inbound",
              "priority":"4093"
            }
          },
          {
            "name":"apiLoadBalancerRule",
            "properties":{
              "description":"Allow anyone to reach the kubernetes API.",
              "protocol":"tcp",
              "sourcePortRange":"*",
              "destinationPortRange":"[string(parameters('kubernetesAPISecurePort'))]",
              "sourceAddressPrefix":"*",
              "destinationAddressPrefix":"[parameters('masterSubnetCidr')]",
              "access":"Allow",
              "direction":"Inbound",
              "priority":"3903"
            }
          },
          {
            "name":"allowEtcdLoadBalancer",
            "properties":{
              "description":"Allow traffic from LB to master instance.",
              "protocol":"tcp",
              "sourcePortRange":"*",
              "destinationPortRange":"[variables('etcdPort')]",
              "sourceAddressPrefix":"AzureLoadBalancer",
              "destinationAddressPrefix":"[parameters('masterSubnetCidr')]",
              "access":"Allow",
              "direction":"Inbound",
              "priority":"3902"
            }
          },
          {
            "name":"etcdLoadBalancerRuleHost",
            "properties":{
              "description":"Allow host cluster to reach the etcd loadbalancer.",
              "protocol":"tcp",
              "sourcePortRange":"*",
              "destinationPortRange":"[variables('etcdPort')]",
              "sourceAddressPrefix":"[parameters('hostClusterCidr')]",
              "destinationAddressPrefix":"[parameters('masterSubnetCidr')]",
              "access":"Allow",
              "direction":"Inbound",
              "priority":"3901"
            }
          },
          {
            "name":"etcdLoadBalancerRuleCluster",
            "properties":{
              "description":"Allow cluster subnet to reach the etcd loadbalancer.",
              "protocol":"tcp",
              "sourcePortRange":"*",
              "destinationPortRange":"[variables('etcdPort')]",
              "sourceAddressPrefix":"[parameters('virtualNetworkCidr')]",
              "destinationAddressPrefix":"[parameters('masterSubnetCidr')]",
              "access":"Allow",
              "direction":"Inbound",
              "priority":"3900"
            }
          },
          {
            "name":"allowWorkerSubnet",
            "properties":{
              "description":"Allow the worker machines to reach the master machines on any ports.",
              "protocol":"*",
              "sourcePortRange":"*",
              "destinationPortRange":"*",
              "sourceAddressPrefix":"[parameters('workerSubnetCidr')]",
              "destinationAddressPrefix":"[parameters('virtualNetworkCidr')]",
              "access":"Allow",
              "direction":"Inbound",
              "priority":"3800"
            }
          },
          {
            "name":"allowCalicoSubnet",
            "properties":{
              "description":"Allow pods to reach the master machines on any ports.",
              "protocol":"*",
              "sourcePortRange":"*",
              "destinationPortRange":"*",
              "sourceAddressPrefix":"[parameters('calicoSubnetCidr')]",
              "destinationAddressPrefix":"[parameters('virtualNetworkCidr')]",
              "access":"Allow",
              "direction":"Inbound",
              "priority":"3700"
            }
          },
          {
            "name":"allowCadvisor",
            "properties":{
              "description":"Allow host cluster Prometheus to reach Cadvisors.",
              "protocol":"tcp",
              "sourcePortRange":"*",
              "destinationPortRange":"[variables('cadvisorPort')]",
              "sourceAddressPrefix":"[parameters('hostClusterCidr')]",
              "destinationAddressPrefix":"[parameters('masterSubnetCidr')]",
              "access":"Allow",
              "direction":"Inbound",
              "priority":"3500"
            }
          },
          {
            "name":"allowKubelet",
            "properties":{
              "description":"Allow host cluster Prometheus to reach Kubelets.",
              "protocol":"tcp",
              "sourcePortRange":"*",
              "destinationPortRange":"[variables('kubeletPort')]",
              "sourceAddressPrefix":"[parameters('hostClusterCidr')]",
              "destinationAddressPrefix":"[parameters('masterSubnetCidr')]",
              "access":"Allow",
              "direction":"Inbound",
              "priority":"3501"
            }
          },
          {
            "name":"allowNodeExporter",
            "properties":{
              "description":"Allow host cluster Prometheus to reach node-exporters.",
              "protocol":"tcp",
              "sourcePortRange":"*",
              "destinationPortRange":"[variables('nodeExporterPort')]",
              "sourceAddressPrefix":"[parameters('hostClusterCidr')]",
              "destinationAddressPrefix":"[parameters('masterSubnetCidr')]",
              "access":"Allow",
              "direction":"Inbound",
              "priority":"3502"
            }
          }
        ]
      }
    },
    {
      "type":"Microsoft.Network/networkSecurityGroups",
      "name":"[variables('workerSecurityGroupName')]",
      "condition":"[equals(parameters('initialProvisioning'), 'Yes')]",
      "apiVersion":"[parameters('networkSecurityGroupsAPIVersion')]",
      "location":"[resourceGroup().location]",
      "tags":{
        "provider":"[toUpper(parameters('GiantSwarmTags').provider)]"
      },
      "properties":{
        "securityRules":[
          {
            "name":"defaultInboundRule",
            "properties":{
              "description":"Default rule that denies any inbound traffic.",
              "protocol":"*",
              "sourcePortRange":"*",
              "destinationPortRange":"*",
              "sourceAddressPrefix":"*",
              "destinationAddressPrefix":"*",
              "access":"Deny",
              "direction":"Inbound",
              "priority":"4096"
            }
          },
          {
            "name":"defaultOutboundRule",
            "properties":{
              "description":"Default rule that allows any outbound traffic.",
              "protocol":"*",
              "sourcePortRange":"*",
              "destinationPortRange":"*",
              "sourceAddressPrefix":"*",
              "destinationAddressPrefix":"*",
              "access":"Allow",
              "direction":"Outbound",
              "priority":"4095"
            }
          },
          {
            "name":"defaultInClusterRule",
            "properties":{
              "description":"Default rule that allows any traffic within the worker subnet.",
              "protocol":"*",
              "sourcePortRange":"*",
              "destinationPortRange":"*",
              "sourceAddressPrefix":"[parameters('workerSubnetCidr')]",
              "destinationAddressPrefix":"[parameters('virtualNetworkCidr')]",
              "access":"Allow",
              "direction":"Inbound",
              "priority":"4094"
            }
          },
          {
            "name":"sshHostClusterToWorkerSubnetRule",
            "properties":{
              "description":"Allow the host cluster machines to reach this guest cluster worker subnet.",
              "protocol":"*",
              "sourcePortRange":"*",
              "destinationPortRange":"22",
              "sourceAddressPrefix":"[parameters('hostClusterCidr')]",
              "destinationAddressPrefix":"[parameters('workerSubnetCidr')]",
              "access":"Allow",
              "direction":"Inbound",
              "priority":"4093"
            }
          },
          {
            "name":"azureLoadBalancerHealthChecks",
            "properties":{
              "description":"Allow Azure Load Balancer health checks.",
              "protocol":"*",
              "sourcePortRange":"*",
              "destinationPortRange":"*",
              "sourceAddressPrefix":"AzureLoadBalancer",
              "destinationAddressPrefix":"*",
              "access":"Allow",
              "direction":"Inbound",
              "priority":"4000"
            }
          },
          {
            "name":"allowMasterSubnet",
            "properties":{
              "description":"Allow the master machines to reach the worker machines on any ports.",
              "protocol":"*",
              "sourcePortRange":"*",
              "destinationPortRange":"*",
              "sourceAddressPrefix":"[parameters('masterSubnetCidr')]",
              "destinationAddressPrefix":"[parameters('virtualNetworkCidr')]",
              "access":"Allow",
              "direction":"Inbound",
              "priority":"3700"
            }
          },
          {
            "name":"allowCalicoSubnet",
            "properties":{
              "description":"Allow pods to reach the worker machines on any ports.",
              "protocol":"*",
              "sourcePortRange":"*",
              "destinationPortRange":"*",
              "sourceAddressPrefix":"[parameters('calicoSubnetCidr')]",
              "destinationAddressPrefix":"[parameters('virtualNetworkCidr')]",
              "access":"Allow",
              "direction":"Inbound",
              "priority":"3600"
            }
          },
          {
            "name":"allowCadvisor",
            "properties":{
              "description":"Allow host cluster Prometheus to reach Cadvisors.",
              "protocol":"tcp",
              "sourcePortRange":"*",
              "destinationPortRange":"[variables('cadvisorPort')]",
              "sourceAddressPrefix":"[parameters('hostClusterCidr')]",
              "destinationAddressPrefix":"[parameters('workerSubnetCidr')]",
              "access":"Allow",
              "direction":"Inbound",
              "priority":"3500"
            }
          },
          {
            "name":"allowKubelet",
            "properties":{
              "description":"Allow host cluster Prometheus to reach Kubelets.",
              "protocol":"tcp",
              "sourcePortRange":"*",
              "destinationPortRange":"[variables('kubeletPort')]",
              "sourceAddressPrefix":"[parameters('hostClusterCidr')]",
              "destinationAddressPrefix":"[parameters('workerSubnetCidr')]",
              "access":"Allow",
              "direction":"Inbound",
              "priority":"3501"
            }
          },
          {
            "name":"allowNodeExporter",
            "properties":{
              "description":"Allow host cluster Prometheus to reach node-exporters.",
              "protocol":"tcp",
              "sourcePortRange":"*",
              "destinationPortRange":"[variables('nodeExporterPort')]",
              "sourceAddressPrefix":"[parameters('hostClusterCidr')]",
              "destinationAddressPrefix":"[parameters('workerSubnetCidr')]",
              "access":"Allow",
              "direction":"Inbound",
              "priority":"3502"
            }
          },
          {
            "name":"allowKubeStateMetrics",
            "properties":{
              "description":"Allow host cluster Prometheus to reach kube-state-metrics.",
              "protocol":"tcp",
              "sourcePortRange":"*",
              "destinationPortRange":"[variables('kubeStateMetricsPort')]",
              "sourceAddressPrefix":"[parameters('hostClusterCidr')]",
              "destinationAddressPrefix":"[parameters('workerSubnetCidr')]",
              "access":"Allow",
              "direction":"Inbound",
              "priority":"3503"
            }
          }
        ]
      }
    }
  ],
  "outputs":{
    "masterSecurityGroupID":{
      "type":"string",
      "value":"[resourceId('Microsoft.Network/networkSecurityGroups', variables('masterSecurityGroupName'))]"
    },
    "workerSecurityGroupID":{
      "type":"string",
      "value":"[resourceId('Microsoft.Network/networkSecurityGroups', variables('workerSecurityGroupName'))]"
    }
  }
}
`

const routeTable string = `{
  "$schema":"http://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
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
    "initialProvisioning":{
      "type":"string",
      "defaultValue":"Yes",
      "metadata":{
        "description":"Whether the deployment is provisioned the very first time."
      }
    },
    "routeTablesAPIVersion":{
      "type":"string",
      "defaultValue":"2016-09-01",
      "metadata":{
        "description":"API version used by the Microsoft.Network/routeTables resource."
      }
    }
  },
  "variables":{
    "name":"[concat(parameters('clusterID'), '-RouteTable')]",
    "id":"[resourceId('Microsoft.Network/routeTables', variables('name'))]"
  },
  "resources":[
    {
      "type":"Microsoft.Network/routeTables",
      "name":"[variables('name')]",
      "condition":"[equals(parameters('initialProvisioning'), 'Yes')]",
      "apiVersion":"[parameters('routeTablesAPIVersion')]",
      "location":"[resourceGroup().location]",
      "tags":{
        "provider":"[toUpper(parameters('GiantSwarmTags').provider)]"
      }
    }
  ],
  "outputs":{
    "name":{
      "type":"string",
      "value":"[variables('name')]"
    },
    "id":{
      "type":"string",
      "value":"[variables('id')]"
    }
  }
}
`

const virtualNetwork string = `{
  "$schema":"https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
  "contentVersion":"1.0.0.0",
  "parameters":{
    "GiantSwarmTags":{
      "type":"object",
      "defaultValue":{
        "provider":"F80D01C0-7AAC-4440-98F6-5061511962AD"
      }
    },
    "virtualNetworksAPIVersion":{
      "type":"string",
      "defaultValue":"2016-12-01",
      "metadata":{
        "description":"API version used by the Microsoft.Network/virtualNetworks resource."
      }
    },
    "virtualNetworkName":{
      "type":"string"
    },
    "virtualNetworkCidr":{
      "type":"string"
    },
    "masterSubnetCidr":{
      "type":"string"
    },
    "workerSubnetCidr":{
      "type":"string"
    },
    "vnetGatewaySubnetName":{
      "type":"string"
    },
    "vpnSubnetCidr":{
      "type":"string"
    },
    "masterSecurityGroupID":{
      "type":"string"
    },
    "workerSecurityGroupID":{
      "type":"string"
    },
    "routeTableID":{
      "type":"string"
    }
  },
  "variables":{
    "virtualNetworkID":"[resourceId('Microsoft.Network/virtualNetworks', parameters('virtualNetworkName'))]",
    "masterSubnetName":"[concat(parameters('virtualNetworkName'), '-MasterSubnet')]",
    "masterSubnetID":"[concat(variables('virtualNetworkID'), '/subnets/', variables('masterSubnetName'))]",
    "workerSubnetName":"[concat(parameters('virtualNetworkName'), '-WorkerSubnet')]",
    "workerSubnetID":"[concat(variables('virtualNetworkID'), '/subnets/', variables('workerSubnetName'))]"
  },
  "resources":[
    {
      "type":"Microsoft.Network/virtualNetworks",
      "name":"[parameters('virtualNetworkName')]",
      "apiVersion":"[parameters('virtualNetworksAPIVersion')]",
      "location":"[resourceGroup().location]",
      "tags":{
        "provider":"[toUpper(parameters('GiantSwarmTags').provider)]"
      },
      "properties":{
        "addressSpace":{
          "addressPrefixes":[
            "[parameters('virtualNetworkCidr')]"
          ]
        },
        "subnets":[
          {
            "name":"[variables('masterSubnetName')]",
            "properties":{
              "addressPrefix":"[parameters('masterSubnetCidr')]",
              "networkSecurityGroup":{
                "id":"[parameters('masterSecurityGroupID')]"
              },
              "routeTable":{
                "id":"[parameters('routeTableID')]"
              }
            }
          },
          {
            "name":"[variables('workerSubnetName')]",
            "properties":{
              "addressPrefix":"[parameters('workerSubnetCidr')]",
              "networkSecurityGroup":{
                "id":"[parameters('workerSecurityGroupID')]"
              },
              "routeTable":{
                "id":"[parameters('routeTableID')]"
              },
              "serviceEndpoints": [
                { "service": "Microsoft.Storage" },
                { "service": "Microsoft.Sql" },
                { "service": "Microsoft.AzureCosmosDB" },
                { "service": "Microsoft.KeyVault" },
                { "service": "Microsoft.ServiceBus" },
                { "service": "Microsoft.EventHub" },
                { "service": "Microsoft.AzureActiveDirectory" },
                { "service": "Microsoft.ContainerRegistry" },
                { "service": "Microsoft.Web" }
              ]
            }
          },
          {
            "name":"[parameters('vnetGatewaySubnetName')]",
            "properties":{
              "addressPrefix":"[parameters('vpnSubnetCidr')]"
            }
          }
        ]
      }
    }
  ],
  "outputs":{
    "masterSubnetID":{
      "type":"string",
      "value":"[variables('masterSubnetID')]"
    },
    "workerSubnetID":{
      "type":"string",
      "value":"[variables('workerSubnetID')]"
    }
  }
}`

const publicLoadBalancer string = `{
  "$schema": "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
  "contentVersion": "1.0.0.0",
  "parameters": {
    "publicIPAddressesAPIVersion": {
      "defaultValue": "2018-12-01",
      "type": "String",
      "metadata": {
        "description": "API version used by the Microsoft.Network/publicIPAddresses resource."
      }
    },
    "loadBalancersAPIVersion": {
      "defaultValue": "2018-12-01",
      "type": "String",
      "metadata": {
        "description": "API version used by the Microsoft.Network/loadBalancers resource."
      }
    },
    "clusterID": {
      "type": "String"
    },
    "GiantSwarmTags": {
      "defaultValue": {
        "provider": "F80D01C0-7AAC-4440-98F6-5061511962AD"
      },
      "type": "Object"
    },
    "prefix": {
      "type": "String"
    },
    "ports": {
      "type": "Array"
    }
  },
  "variables": {
    "loadBalancerPublicIPName": "[concat(parameters('clusterID'), '-', parameters('prefix'), '-PublicLoadBalancer-PublicIP')]",
    "loadBalancerPublicIPId": "[resourceId('Microsoft.Network/publicIPAddresses', variables('LoadBalancerPublicIPName'))]",
    "loadBalancerName": "[concat(parameters('clusterID'), '-', parameters('prefix'), '-PublicLoadBalancer')]",
    "loadBalancerSkuName": "Standard",
    "loadBalancerID": "[resourceId('Microsoft.Network/loadBalancers', variables('loadBalancerName'))]",
    "loadBalancerBackendPoolName": "[concat(parameters('clusterID'), '-', parameters('prefix'), '-', 'PublicLoadBalancer-BackendPool')]",
    "loadBalancerBackendPoolID": "[concat(variables('loadBalancerID'),'/backendAddressPools/',variables('loadBalancerBackendPoolName'))]",
    "loadBalancerFrontendName": "[concat(parameters('clusterID'), '-', parameters('prefix'), '-', 'PublicLoadBalancer-Frontend')]",
    "loadBalancerFrontendID": "[concat(variables('loadBalancerID'),'/frontendIPConfigurations/',variables('loadBalancerFrontendName'))]",
    "loadBalancerProbeName": "[concat(parameters('clusterID'), '-', parameters('prefix'), '-', 'PublicLoadBalancer-Probe')]",
    "loadBalancerProbeID": "[concat(variables('loadBalancerID'),'/probes/',variables('loadBalancerProbeName'))]"
  },
  "resources": [
    {
      "type": "Microsoft.Network/publicIPAddresses",
      "apiVersion": "[parameters('publicIPAddressesAPIVersion')]",
      "name": "[variables('loadBalancerPublicIPName')]",
      "location": "[resourceGroup().location]",
      "tags": {
        "provider": "[toUpper(parameters('GiantSwarmTags').provider)]"
      },
      "sku": {
        "name": "[variables('loadBalancerSkuName')]"
      },
      "properties": {
        "publicIPAllocationMethod": "Static",
        "publicIPAddressVersion": "IPv4"
      }
    },
    {
      "type": "Microsoft.Network/loadBalancers",
      "apiVersion": "[parameters('loadBalancersAPIVersion')]",
      "name": "[variables('loadBalancerName')]",
      "location": "[resourceGroup().location]",
      "dependsOn": [
        "[concat('Microsoft.Network/publicIPAddresses/', variables('loadBalancerPublicIPName'))]"
      ],
      "tags": {
        "provider": "[toUpper(parameters('GiantSwarmTags').provider)]"
      },
      "sku": {
        "name": "[variables('loadBalancerSkuName')]"
      },
      "properties": {
        "frontendIPConfigurations": [
          {
            "name": "[variables('loadBalancerFrontendName')]",
            "properties": {
              "publicIPAddress": {
                "id": "[variables('loadBalancerPublicIPId')]"
              }
            }
          }
        ],
        "backendAddressPools": [
          {
            "name": "[variables('loadBalancerBackendPoolName')]"
          }
        ],
        "copy": [
          {
            "name": "loadBalancingRules",
            "count": "[length(parameters('ports'))]",
            "input": {
              "name": "[concat('loadBalancingRule-', string(add(copyIndex('loadBalancingRules'), 1)))]",
              "properties": {
                "frontendIPConfiguration": {
                  "id": "[variables('loadBalancerFrontendID')]"
                },
                "backendAddressPool": {
                  "id": "[variables('loadBalancerBackendPoolID')]"
                },
                "protocol": "[parameters('ports')[copyIndex('loadBalancingRules')].protocol]",
                "frontendPort": "[parameters('ports')[copyIndex('loadBalancingRules')].frontend]",
                "backendPort": "[parameters('ports')[copyIndex('loadBalancingRules')].backend]",
                "enableFloatingIP": false,
                "probe": {
                  "id": "[concat(variables('loadBalancerProbeID'), '-', string(add(copyIndex('loadBalancingRules'), 1)))]"
                }
              }
            }
          },
          {
            "name": "probes",
            "count": "[length(parameters('ports'))]",
            "input": {
              "name": "[concat(variables('loadBalancerProbeName'), '-', string(add(copyIndex('probes'), 1)))]",
              "properties": {
                "protocol": "[parameters('ports')[copyIndex('probes')].probeProtocol]",
                "port": "[parameters('ports')[copyIndex('probes')].probePort]",
                "intervalInSeconds": 5,
                "numberOfProbes": 2
              }
            }
          }
        ]
      }
    }
  ],
  "outputs": {
    "ipAddress": {
      "type": "String",
      "value": "[reference(variables('loadBalancerPublicIPId'), parameters('publicIPAddressesAPIVersion')).ipAddress]"
    },
    "backendPoolId": {
      "type": "String",
      "value": "[reference(variables('loadBalancerID')).backendAddressPools[0].id]"
    }
  }
}`

const privateLoadBalancer string = `{
  "$schema":"https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
  "contentVersion":"1.0.0.0",
  "parameters":{
    "loadBalancersAPIVersion":{
      "type":"string",
      "defaultValue":"2018-12-01",
      "metadata":{
        "description":"API version used by the Microsoft.Network/loadBalancers resource."
      }
    },
    "clusterID":{
      "type":"string"
    },
    "GiantSwarmTags":{
      "type":"object",
      "defaultValue":{
        "provider":"F80D01C0-7AAC-4440-98F6-5061511962AD"
      }
    },
    "prefix":{
      "type":"string"
    },
    "ports":{
      "type":"array"
    },
    "masterSubnetID":{
      "type":"string"
    }
  },
  "variables":{
    "loadBalancerName":"[concat(parameters('clusterID'), '-', parameters('prefix'), '-PrivateLoadBalancer')]",
    "loadBalancerSkuName": "Standard",
    "loadBalancerID":"[resourceId('Microsoft.Network/loadBalancers', variables('loadBalancerName'))]",
    "loadBalancerBackendPoolName":"[concat(parameters('clusterID'), '-', parameters('prefix'), '-', 'PrivateLoadBalancer-BackendPool')]",
    "loadBalancerBackendPoolID":"[concat(variables('loadBalancerID'),'/backendAddressPools/',variables('loadBalancerBackendPoolName'))]",
    "loadBalancerFrontendName":"[concat(parameters('clusterID'), '-', parameters('prefix'), '-', 'PrivateLoadBalancer-Frontend')]",
    "loadBalancerFrontendID":"[concat(variables('loadBalancerID'),'/frontendIPConfigurations/',variables('loadBalancerFrontendName'))]",
    "loadBalancerProbeName":"[concat(parameters('clusterID'), '-', parameters('prefix'), '-', 'PrivateLoadBalancer-Probe')]",
    "loadBalancerProbeID":"[concat(variables('loadBalancerID'),'/probes/',variables('loadBalancerProbeName'))]",
    "loadBalancerSubnetID":"[parameters('masterSubnetID')]"
  },
  "resources":[
    {
      "apiVersion":"[parameters('loadBalancersAPIVersion')]",
      "name":"[variables('loadBalancerName')]",
      "type":"Microsoft.Network/loadBalancers",
      "location":"[resourceGroup().location]",
      "sku": {
        "name": "[variables('loadBalancerSkuName')]"
      },
      "tags":{
        "provider":"[toUpper(parameters('GiantSwarmTags').provider)]"
      },
      "properties":{
        "frontendIPConfigurations":[
          {
            "name":"[variables('loadBalancerFrontendName')]",
            "properties":{
              "privateIPAllocationMethod":"Dynamic",
              "subnet":{
                "id":"[variables('loadBalancerSubnetID')]"
              }
            }
          }
        ],
        "backendAddressPools":[
          {
            "name":"[variables('loadBalancerBackendPoolName')]"
          }
        ],
        "copy":[
          {
            "name":"loadBalancingRules",
            "count":"[length(parameters('ports'))]",
            "input":{
              "name":"[concat('loadBalancingRule-', string(add(copyIndex('loadBalancingRules'), 1)))]",
              "properties":{
                "frontendIPConfiguration":{
                  "id":"[variables('loadBalancerFrontendID')]"
                },
                "backendAddressPool":{
                  "id":"[variables('loadBalancerBackendPoolID')]"
                },
                "protocol":"Tcp",
                "frontendPort":"[parameters('ports')[copyIndex('loadBalancingRules')].frontend]",
                "backendPort":"[parameters('ports')[copyIndex('loadBalancingRules')].backend]",
                "enableFloatingIP":false,
                "probe":{
                  "id":"[concat(variables('loadBalancerProbeID'), '-', string(add(copyIndex('loadBalancingRules'), 1)))]"
                }
              }
            }
          },
          {
            "name":"probes",
            "count":"[length(parameters('ports'))]",
            "input":{
              "name":"[concat(variables('loadBalancerProbeName'), '-', string(add(copyIndex('probes'), 1)))]",
              "properties":{
                "protocol":"Tcp",
                "port":"[parameters('ports')[copyIndex('probes')].backend]",
                "intervalInSeconds":5,
                "numberOfProbes":2
              }
            }
          }
        ]
      }
    }
  ],
  "outputs":{
    "ipAddress":{
      "type":"string",
      "value":"[reference(variables('loadBalancerID')).frontendIPConfigurations[0].properties.privateIPAddress]"
    },
    "backendPoolId":{
      "type":"string",
      "value":"[reference(variables('loadBalancerID')).backendAddressPools[0].id]"
    }
  }
}`

const kubernetesLoadBalanacer string = `{
  "$schema":"https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
  "contentVersion":"1.0.0.0",
  "parameters":{
    "initialProvisioning":{
      "type":"string",
      "defaultValue":"Yes",
      "metadata":{
        "description":"Whether the deployment is provisioned the very first time."
      }
    }
  },
  "variables":{
    "loadBalancerSkuName": "Standard"
  },
  "resources":[
    {
      "apiVersion":"2017-10-01",
      "type":"Microsoft.Network/publicIPAddresses",
      "name":"dummy-pip",
      "sku": {
        "name": "[variables('loadBalancerSkuName')]"
      },
      "location":"[resourceGroup().location]",
      "properties":{
        "publicIPAllocationMethod":"Static"
      }
    },
    {
      "apiVersion":"2017-10-01",
      "name":"kubernetes",
      "type":"Microsoft.Network/loadBalancers",
      "condition":"[equals(parameters('initialProvisioning'), 'Yes')]",
      "location":"[resourceGroup().location]",
      "sku": {
        "name": "[variables('loadBalancerSkuName')]"
      },
      "dependsOn":[
        "[concat('Microsoft.Network/publicIPAddresses/', 'dummy-pip')]"
      ],
      "properties":{
        "frontendIPConfigurations":[
          {
            "name":"dummy-frontend",
            "properties":{
              "publicIPAddress":{
                "id":"[resourceId('Microsoft.Network/publicIPAddresses', 'dummy-pip')]"
              }
            }
          }
        ],
        "backendAddressPools":[
          {
            "name":"kubernetes"
          }
        ]
      }
    }
  ]
}`

const dnsA string = `{
  "$schema":"https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
  "contentVersion":"1.0.0.0",
  "parameters":{
    "dnsZone":{
      "type":"string"
    },
    "GiantSwarmTags":{
      "type":"object",
      "defaultValue":{
        "provider":"F80D01C0-7AAC-4440-98F6-5061511962AD"
      }
    },
    "prefix":{
      "type":"string"
    },
    "ipAddress":{
      "type":"string"
    }
  },
  "resources":[
    {
      "type":"Microsoft.Network/dnszones",
      "name":"[parameters('dnsZone')]",
      "apiVersion":"2016-04-01",
      "location":"global",
      "properties":{

      },
      "tags":{
        "provider":"[toUpper(parameters('GiantSwarmTags').provider)]"
      }
    },
    {
      "type":"Microsoft.Network/dnszones/a",
      "name":"[concat(parameters('dnsZone'), '/', parameters('prefix'))]",
      "apiVersion":"2016-04-01",
      "location":"global",
      "tags":{
        "provider":"[toUpper(parameters('GiantSwarmTags').provider)]"
      },
      "dependsOn":[
        "[parameters('dnsZone')]"
      ],
      "properties":{
        "TTL":3600,
        "ARecords":[
          {
            "ipv4Address":"[parameters('ipAddress')]"
          }
        ]
      }
    }
  ]
}`

const containerSetup string = `{
    "$schema": "http://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
    "contentVersion": "1.0.0.0",
    "parameters": {
        "blobContainerName": {
            "type": "string"
        },
        "clusterID":{
          "type":"string"
        },
        "GiantSwarmTags":{
          "type":"object",
          "defaultValue":{
            "provider":"F80D01C0-7AAC-4440-98F6-5061511962AD"
          }
        },
        "storageAccountName":{
          "type":"string"
        }
    },
    "variables": {
        "accessTier": "Cool",
        "accountType": "Standard_RAGRS",
        "kind": "BlobStorage",
        "supportsHttpsTrafficOnly": true
    },
    "resources": [
        {
            "name": "[parameters('storageAccountName')]",
            "type": "Microsoft.Storage/storageAccounts",
            "apiVersion": "2018-07-01",
            "location":"[resourceGroup().location]",
            "properties": {
                "accessTier": "[variables('accessTier')]",
                "accountType": "[variables('accountType')]",
                "supportsHttpsTrafficOnly": "[variables('supportsHttpsTrafficOnly')]"
            },
            "sku": {
                "name": "[variables('accountType')]"
            },
            "kind": "[variables('kind')]",
            "tags": {
                "provider":"[toUpper(parameters('GiantSwarmTags').provider)]"
            },
            "resources": [
                {
                "name": "[concat('default/',parameters('blobContainerName'))]",
                "type": "blobServices/containers",
                "apiVersion": "2018-03-01-preview",
                "dependsOn": [
                    "[parameters('storageAccountName')]"
                ]
                }
            ]
        }
    ],
    "outputs": {}
}`

const dnsCname string = `{
  "$schema":"https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
  "contentVersion":"1.0.0.0",
  "parameters":{
    "dnsZone":{
      "type":"string"
    },
    "GiantSwarmTags":{
      "type":"object",
      "defaultValue":{
        "provider":"F80D01C0-7AAC-4440-98F6-5061511962AD"
      }
    },
    "prefix":{
      "type":"string"
    },
    "cname":{
      "type":"string"
    }
  },
  "resources":[
    {
      "type":"Microsoft.Network/dnszones",
      "name":"[parameters('dnsZone')]",
      "apiVersion":"2016-04-01",
      "location":"global",
      "properties":{

      },
      "tags":{
        "provider":"[toUpper(parameters('GiantSwarmTags').provider)]"
      }
    },
    {
      "type":"Microsoft.Network/dnszones/cname",
      "name":"[concat(parameters('dnsZone'), '/', parameters('prefix'))]",
      "apiVersion":"2016-04-01",
      "location":"global",
      "tags":{
        "provider":"[toUpper(parameters('GiantSwarmTags').provider)]"
      },
      "dependsOn":[
        "[parameters('dnsZone')]"
      ],
      "properties":{
        "TTL":3600,
        "CNAMERecord":{
          "cname":"[parameters('cname')]"
        }
      }
    }
  ]
}`

const main string = `{
  "$schema":"https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
  "contentVersion":"1.0.0.0",
  "parameters":{
    "blobContainerName":{
      "type":"string",
      "metadata":{
        "description":"Container name for the ignition."
      }
    },
    "clusterID":{
      "type":"string",
      "metadata":{
        "description":"Unique ID of this guest cluster."
      }
    },
    "GiantSwarmTags":{
      "type":"object",
      "defaultValue":{
        "provider":"F80D01C0-7AAC-4440-98F6-5061511962AD"
      }
    },
    "initialProvisioning":{
      "type":"string",
      "defaultValue":"Yes",
      "metadata":{
        "description":"Whether the deployment is provisioned the very first time."
      }
    },
    "storageAccountName":{
      "type":"string",
      "metadata":{
        "description":"Storage account name for the ignition."
      }
    },
    "virtualNetworkName":{
      "type":"string",
      "metadata":{
        "description":"Virtual network's name of the guest cluster"
      }
    },
    "virtualNetworkCidr":{
      "type":"string",
      "metadata":{
        "description":"The main CIDR block reserved for this virtual network."
      }
    },
    "vnetGatewaySubnetName": {
      "type":"string"
    },
    "calicoSubnetCidr":{
      "type":"string"
    },
    "masterSubnetCidr":{
      "type":"string"
    },
    "workerSubnetCidr":{
      "type":"string"
    },
    "vpnSubnetCidr":{
      "type":"string"
    },
    "hostClusterCidr":{
      "type":"string"
    },
    "kubernetesAPISecurePort":{
      "type":"int"
    },
    "dnsZones":{
      "type":"object",
      "metadata":{
        "description":"The DNS zones for kubernetes api and ingress."
      }
    }
  },
  "variables":{
    "masterPrefix":"Master",
    "workerPrefix":"Worker"
  },
  "resources":[
    {
      "apiVersion":"2016-09-01",
      "name":"security_groups_setup",
      "type":"Microsoft.Resources/deployments",
      "properties":{
        "expressionEvaluationOptions": {
          "scope": "inner"
        },
        "mode":"incremental",
        "template": %s,
        "parameters":{
          "clusterID":{
            "value":"[parameters('clusterID')]"
          },
          "GiantSwarmTags":{
            "value":"[parameters('GiantSwarmTags')]"
          },
          "virtualNetworkCidr":{
            "value":"[parameters('virtualNetworkCidr')]"
          },
          "calicoSubnetCidr":{
            "value":"[parameters('calicoSubnetCidr')]"
          },
          "masterSubnetCidr":{
            "value":"[parameters('masterSubnetCidr')]"
          },
          "workerSubnetCidr":{
            "value":"[parameters('workerSubnetCidr')]"
          },
          "hostClusterCidr":{
            "value":"[parameters('hostClusterCidr')]"
          },
          "kubernetesAPISecurePort":{
            "value":"[parameters('kubernetesAPISecurePort')]"
          },
          "initialProvisioning":{
            "value":"[parameters('initialProvisioning')]"
          }
        }
      }
    },
    {
      "apiVersion":"2016-09-01",
      "name":"route_table_setup",
      "type":"Microsoft.Resources/deployments",
      "properties":{
        "expressionEvaluationOptions": {
          "scope": "inner"
        },
        "mode":"incremental",
        "template": %s,
        "parameters":{
          "clusterID":{
            "value":"[parameters('clusterID')]"
          },
          "GiantSwarmTags":{
            "value":"[parameters('GiantSwarmTags')]"
          },
          "initialProvisioning":{
            "value":"[parameters('initialProvisioning')]"
          }
        }
      }
    },
    {
      "apiVersion":"2016-09-01",
      "name":"virtual_network_setup",
      "type":"Microsoft.Resources/deployments",
      "dependsOn":[
        "route_table_setup"
      ],
      "properties":{
        "expressionEvaluationOptions": {
          "scope": "inner"
        },
        "mode":"incremental",
        "template": %s,
        "parameters":{
          "GiantSwarmTags":{
            "value":"[parameters('GiantSwarmTags')]"
          },
          "virtualNetworkName":{
            "value":"[parameters('virtualNetworkName')]"
          },
          "virtualNetworkCidr":{
            "value":"[parameters('virtualNetworkCidr')]"
          },
          "masterSubnetCidr":{
            "value":"[parameters('masterSubnetCidr')]"
          },
          "workerSubnetCidr":{
            "value":"[parameters('workerSubnetCidr')]"
          },
          "vnetGatewaySubnetName":{
            "value":"[parameters('vnetGatewaySubnetName')]"
          },
          "vpnSubnetCidr":{
            "value":"[parameters('vpnSubnetCidr')]"
          },
          "masterSecurityGroupID":{
            "value":"[reference('security_groups_setup').outputs.masterSecurityGroupID.value]"
          },
          "workerSecurityGroupID":{
            "value":"[reference('security_groups_setup').outputs.workerSecurityGroupID.value]"
          },
          "routeTableID":{
            "value":"[reference('route_table_setup').outputs.id.value]"
          }
        }
      }
    },
    {
      "apiVersion":"2016-09-01",
      "name":"api_load_balancer_setup",
      "type":"Microsoft.Resources/deployments",
      "properties":{
        "expressionEvaluationOptions": {
          "scope": "inner"
        },
        "mode":"incremental",
        "template": %s,
        "parameters":{
          "clusterID":{
            "value":"[parameters('clusterID')]"
          },
          "GiantSwarmTags":{
            "value":"[parameters('GiantSwarmTags')]"
          },
          "prefix":{
            "value":"API"
          },
          "ports":{
            "value":[
              {
                "protocol":"Tcp",
                "frontend":443,
                "backend":443,
                "probeProtocol": "Tcp",
                "probePort": 443
              },
              {
                "protocol":"Udp",
                "frontend":60001,
                "backend":60001,
                "probeProtocol": "Tcp",
                "probePort": 443
              }
            ]
          }
        }
      }
    },
    {
      "apiVersion":"2016-09-01",
      "name":"etcd_load_balancer_setup",
      "type":"Microsoft.Resources/deployments",
      "properties":{
        "expressionEvaluationOptions": {
          "scope": "inner"
        },
        "mode":"incremental",
        "template": %s,
        "parameters":{
          "clusterID":{
            "value":"[parameters('clusterID')]"
          },
          "GiantSwarmTags":{
            "value":"[parameters('GiantSwarmTags')]"
          },
          "prefix":{
            "value":"ETCD"
          },
          "ports":{
            "value":[
              {
                "frontend":2379,
                "backend":2379
              }
            ]
          },
          "masterSubnetID":{
            "value":"[reference('virtual_network_setup').outputs.masterSubnetID.value]"
          }
        }
      }
    },
    {
      "apiVersion":"2016-09-01",
      "name":"kubernetes_load_balancer_setup",
      "type":"Microsoft.Resources/deployments",
      "properties":{
        "expressionEvaluationOptions": {
          "scope": "inner"
        },
        "mode":"incremental",
        "template": %s,
        "parameters":{
          "initialProvisioning":{
            "value":"[parameters('initialProvisioning')]"
          }
        }
      }
    },
    {
      "apiVersion":"2016-09-01",
      "name":"kubernetes_api_dns_setup",
      "type":"Microsoft.Resources/deployments",
      "properties":{
        "expressionEvaluationOptions": {
          "scope": "inner"
        },
        "mode":"incremental",
        "template": %s,
        "parameters":{
          "dnsZone":{
            "value":"[concat(parameters('clusterID'), '.k8s.', parameters('dnsZones').api.name)]"
          },
          "GiantSwarmTags":{
            "value":"[parameters('GiantSwarmTags')]"
          },
          "prefix":{
            "value":"api"
          },
          "ipAddress":{
            "value":"[reference('api_load_balancer_setup').outputs.ipAddress.value]"
          }
        }
      }
    },
    {
      "apiVersion":"2016-09-01",
      "name":"kubernetes_etcd_dns_setup",
      "type":"Microsoft.Resources/deployments",
      "properties":{
        "expressionEvaluationOptions": {
          "scope": "inner"
        },
        "mode":"incremental",
        "template": %s,
        "parameters":{
          "dnsZone":{
            "value":"[concat(parameters('clusterID'), '.k8s.', parameters('dnsZones').api.name)]"
          },
          "GiantSwarmTags":{
            "value":"[parameters('GiantSwarmTags')]"
          },
          "prefix":{
            "value":"etcd"
          },
          "ipAddress":{
            "value":"[reference('etcd_load_balancer_setup').outputs.ipAddress.value]"
          }
        }
      }
    },
    {
      "apiVersion":"2016-09-01",
      "name":"container_setup",
      "type":"Microsoft.Resources/deployments",
      "properties":{
        "expressionEvaluationOptions": {
          "scope": "inner"
        },
        "mode":"incremental",
        "template": %s,
        "parameters":{
          "blobContainerName":{
            "value":"[parameters('blobContainerName')]"
          },
          "clusterID":{
            "value":"[parameters('clusterID')]"
          },
          "GiantSwarmTags":{
            "value":"[parameters('GiantSwarmTags')]"
          },
          "storageAccountName":{
            "value":"[parameters('storageAccountName')]"
          }
        }
      }
    },
    {
      "apiVersion":"2016-09-01",
      "name":"kubernetes_ingress_wildcard_dns_setup",
      "type":"Microsoft.Resources/deployments",
      "properties":{
        "expressionEvaluationOptions": {
          "scope": "inner"
        },
        "mode":"incremental",
        "template": %s,
        "parameters":{
          "dnsZone":{
            "value":"[concat(parameters('clusterID'), '.k8s.', parameters('dnsZones').ingress.name)]"
          },
          "GiantSwarmTags":{
            "value":"[parameters('GiantSwarmTags')]"
          },
          "prefix":{
            "value":"*"
          },
          "cname":{
            "value":"[concat('ingress.', parameters('clusterID'), '.k8s.', parameters('dnsZones').ingress.name)]"
          }
        }
      }
    }
  ]
}`

func getARMTemplate() (map[string]interface{}, error) {
	contents := make(map[string]interface{})
	if err := json.Unmarshal([]byte(getARMTemplateAsString()), &contents); err != nil {
		return nil, err
	}
	return contents, nil
}

func getARMTemplateAsString() string {
	return fmt.Sprintf(main, securityGroups, routeTable, virtualNetwork, publicLoadBalancer, privateLoadBalancer, kubernetesLoadBalanacer, dnsA, dnsA, containerSetup, dnsCname)
}