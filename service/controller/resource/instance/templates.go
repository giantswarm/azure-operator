package instance

import (
	"encoding/json"
	"fmt"
)

const vmssTemplate string = `{
  "$schema":"https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
  "contentVersion":"1.0.0.0",
  "parameters":{
    "location":{
      "type":"string"
    },
    "azureOperatorVersion":{
      "type":"string"
    },
    "GiantSwarmTags":{
      "type":"object",
      "defaultValue":{
        "provider":"F80D01C0-7AAC-4440-98F6-5061511962AD"
      }
    },
    "vmssMSIEnabled":{
      "type":"bool"
    },
    "vmssName":{
      "type":"string"
    },
    "vmssVmSize":{
      "type":"string"
    },
    "vmssVmCount":{
      "type":"int",
      "minValue":1,
      "maxValue":100
    },
    "vmssVmDataDisks":{
      "type":"array"
    },
    "vmssSshUser":{
      "type":"string"
    },
    "vmssSshPublicKey":{
      "type":"string"
    },
    "vmssOsImagePublisher":{
      "type":"string"
    },
    "vmssOsImageOffer":{
      "type":"string"
    },
    "vmssOsImageSKU":{
      "type":"string"
    },
    "vmssOsImageVersion":{
      "type":"string"
    },
    "vmssOverprovision":{
      "type":"string",
      "defaultValue":"true"
    },
    "vmssStorageAccountType":{
      "type":"string"
    },
    "vmssVmCustomData":{
      "type":"securestring"
    },
    "vmssLbBackendPools":{
      "type":"array"
    },
    "vmssVnetSubnetId":{
      "type":"string"
    },
    "vmssUpgradePolicy":{
      "type":"string",
      "defaultValue":"Manual"
    },
    "zones":{
      "type":"array",
      "defaultValue": [1],
      "metadata":{
        "description":"Availability zones used to create the cluster."
      }
    }
  },
  "variables":{
    "authorizationAPIVersion":"2017-05-01",
    "computeAPIVersion":"2019-07-01",
    "contributorRoleDefinitionGUID":"b24988ac-6180-42a0-ab88-20f7382dd24c",
    "contributorRoleDefinitionId":"[concat('/subscriptions/', subscription().subscriptionId, '/providers/Microsoft.Authorization/roleDefinitions/', variables('contributorRoleDefinitionGUID'))]",
    "roleAssignmentName":"[guid(concat(parameters('vmssName'), '-', 'roleassignment'))]",
    "vmssExtensions":[
      {
        "name":"MSILinuxExtension",
        "properties":{
          "publisher":"Microsoft.ManagedIdentity",
          "type":"ManagedIdentityExtensionForLinux",
          "typeHandlerVersion":"1.0",
          "autoUpgradeMinorVersion":true,
          "settings":{
            "port":50342
          },
          "protectedSettings":{

          }
        }
      }
    ],
    "vmssResourceId":"[resourceId('Microsoft.Compute/virtualMachineScaleSets', parameters('vmssName'))]",
    "vmssSinglePlacementGroup":"true",
    "vmssVmNamePrefix":"[concat(toLower(parameters('vmssName')), '-')]"
  },
  "resources":[
    {
      "apiVersion":"[variables('computeAPIVersion')]",
      "type":"Microsoft.Compute/virtualMachineScaleSets",
      "name":"[parameters('vmssName')]",
      "location":"[parameters('location')]",
      "zones": "[if(greater(length(parameters('zones')),0), parameters('zones'), json('null'))]",
      "tags":{
        "provider":"[toUpper(parameters('GiantSwarmTags').provider)]",
        "cluster-autoscaler-enabled":"true",
        "cluster-autoscaler-name":"[parameters('vmssName')]",
        "gs-azure-operator.giantswarm.io-version":"[parameters('azureOperatorVersion')]"
      },
      "sku":{
        "name":"[parameters('vmssVmSize')]",
        "tier":"Standard",
        "capacity":"[int(parameters('vmssVmCount'))]"
      },
      "identity":{
        "type":"[if(parameters('vmssMSIEnabled'), 'systemAssigned', 'None')]"
      },
      "properties":{
        "overprovision":"[parameters('vmssOverprovision')]",
        "upgradePolicy":{
          "mode":"[parameters('vmssUpgradePolicy')]"
        },
        "singlePlacementGroup":"[variables('vmssSinglePlacementGroup')]",
        "virtualMachineProfile":{
          "osProfile":{
            "adminUsername":"[parameters('vmssSshUser')]",
            "computerNamePrefix":"[variables('vmssVmNamePrefix')]",
            "customData":"[parameters('vmssVmCustomData')]",
            "linuxConfiguration":{
              "disablePasswordAuthentication":"true",
              "ssh":{
                "publicKeys":[
                  {
                    "keyData":"[parameters('vmssSshPublicKey')]",
                    "path":"[concat('/home/', parameters('vmssSshUser'), '/.ssh/authorized_keys')]"
                  }
                ]
              }
            }
          },
          "storageProfile":{
            "imageReference":{
              "publisher":"[parameters('vmssOsImagePublisher')]",
              "offer":"[parameters('vmssOsImageOffer')]",
              "sku":"[parameters('vmssOsImageSKU')]",
              "version":"[parameters('vmssOsImageVersion')]"
            },
            "osDisk":{
              "caching":"ReadWrite",
              "createOption":"FromImage",
              "managedDisk":{
                "storageAccountType":"[parameters('vmssStorageAccountType')]"
              }
            },
            "dataDisks":"[parameters('vmssVmDataDisks')]"
          },
          "networkProfile":{
            "networkInterfaceConfigurations":[
              {
                "name":"[concat(parameters('vmssName'), '-nic')]",
                "properties":{
                  "enableIPForwarding":true,
                  "primary":"true",
                  "ipConfigurations":[
                    {
                      "name":"[concat(parameters('vmssName'), '-ipconfig')]",
                      "properties":{
                        "subnet":{
                          "id":"[parameters('vmssVnetSubnetId')]"
                        },
                        "loadBalancerBackendAddressPools":"[parameters('vmssLbBackendPools')]"
                      }
                    }
                  ]
                }
              }
            ]
          },
          "extensionProfile":{
            "extensions":"[if(parameters('vmssMSIEnabled'), variables('vmssExtensions'), json('[]'))]"
          }
        }
      }
    },
    {
      "apiVersion":"[variables('authorizationAPIVersion')]",
      "condition":"[parameters('vmssMSIEnabled')]",
      "type":"Microsoft.Authorization/roleAssignments",
      "name":"[variables('roleAssignmentName')]",
      "tags":{
        "provider":"[toUpper(parameters('GiantSwarmTags').provider)]"
      },
      "properties":{
        "roleDefinitionId":"[variables('contributorRoleDefinitionId')]",
        "principalId":"[if(parameters('vmssMSIEnabled'), reference(concat(resourceId('Microsoft.Compute/virtualMachineScaleSets/', parameters('vmssName')),'/providers/Microsoft.ManagedIdentity/Identities/default'),'2015-08-31-PREVIEW').principalId, '')]",
        "scope":"[resourceGroup().id]"
      }
    }
  ]
}`

const main string = `{
  "$schema":"https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
  "contentVersion":"1.0.0.0",
  "parameters":{
    "apiLBBackendPoolID":{
      "type":"string",
      "metadata":{
        "description":"Output value of the API load balancer backend pool ID as referenced from the API load balancer setup."
      }
    },
    "azureOperatorVersion":{
      "type":"string",
      "metadata":{
        "description":"Version of the azure operator that created the deployment."
      }
    },
    "clusterID":{
      "type":"string",
      "metadata":{
        "description":"Unique ID of the guest cluster."
      }
    },
    "etcdLBBackendPoolID":{
      "type":"string",
      "metadata":{
        "description":"Output value of the Etcd load balancer backend pool ID as referenced from the Etcd load balancer setup."
      }
    },
    "GiantSwarmTags":{
      "type":"object",
      "defaultValue":{
        "provider":"F80D01C0-7AAC-4440-98F6-5061511962AD"
      }
    },
    "vmssMSIEnabled":{
      "type":"bool"
    },
    "workerCloudConfigData":{
      "type":"secureString",
      "metadata":{
        "description":"Base64-encoded cloud-config data."
      }
    },
    "workerNodes":{
      "type":"array"
    },
    "workerSubnetID":{
      "type":"string",
      "metadata":{
        "description":"Output value of the worker subnet ID as referenced from the virtual network setup."
      }
    },
    "zones":{
      "type":"array",
      "defaultValue": [1],
      "metadata":{
        "description":"Availability zones used to create the cluster."
      }
    }
  },
  "variables":{
    "apiVersion":"2017-08-01",
    "kubernetesLBBackendID":"[concat(resourceId('Microsoft.Network/loadBalancers', 'kubernetes'), '/backendAddressPools/kubernetes')]",
    "vmssStandardLrsSize":[
      "Standard_A0",
      "Standard_A1",
      "Standard_A10",
      "Standard_A11",
      "Standard_A1_v2",
      "Standard_A2",
      "Standard_A2_v2",
      "Standard_A2m_v2",
      "Standard_A3",
      "Standard_A4",
      "Standard_A4_v2",
      "Standard_A4m_v2",
      "Standard_A5",
      "Standard_A6",
      "Standard_A7",
      "Standard_A8",
      "Standard_A8_v2",
      "Standard_A8m_v2",
      "Standard_A9",
      "Standard_D1",
      "Standard_D11",
      "Standard_D11_v2",
      "Standard_D11_v2_Promo",
      "Standard_D12",
      "Standard_D12_v2",
      "Standard_D12_v2_Promo",
      "Standard_D13",
      "Standard_D13_v2",
      "Standard_D13_v2_Promo",
      "Standard_D14",
      "Standard_D14_v2",
      "Standard_D14_v2_Promo",
      "Standard_D15_v2",
      "Standard_D16_v3",
      "Standard_D1_v2",
      "Standard_D2",
      "Standard_D2_v2",
      "Standard_D2_v2_Promo",
      "Standard_D2_v3",
      "Standard_D3",
      "Standard_D32_v3",
      "Standard_D3_v2",
      "Standard_D3_v2_Promo",
      "Standard_D4",
      "Standard_D4_v2",
      "Standard_D4_v2_Promo",
      "Standard_D4_v3",
      "Standard_D5_v2",
      "Standard_D5_v2_Promo",
      "Standard_D64_v3",
      "Standard_D8_v3",
      "Standard_E16_v3",
      "Standard_E2_v3",
      "Standard_E32_v3",
      "Standard_E4_v3",
      "Standard_E64_v3",
      "Standard_E8_v3",
      "Standard_F1",
      "Standard_F16",
      "Standard_F2",
      "Standard_F4",
      "Standard_F8",
      "Standard_G1",
      "Standard_G2",
      "Standard_G3",
      "Standard_G4",
      "Standard_G5",
      "Standard_H16",
      "Standard_H16m",
      "Standard_H16mr",
      "Standard_H16r",
      "Standard_H8",
      "Standard_H8m",
      "Standard_NC12",
      "Standard_NC24",
      "Standard_NC24r",
      "Standard_NC6",
      "Standard_NV12",
      "Standard_NV24",
      "Standard_NV6"
    ]
  },
  "resources":[
    {
      "apiVersion":"[variables('apiVersion')]",
      "type":"Microsoft.Resources/deployments",
      "name":"worker-vmss-deploy",
      "properties":{
        "expressionEvaluationOptions": {
          "scope": "inner"
        },
        "mode":"incremental",
        "template": %s,
        "parameters":{
          "location":{
            "value":"[resourceGroup().location]"
          },
          "azureOperatorVersion":{
            "value":"[parameters('azureOperatorVersion')]"
          },
          "GiantSwarmTags":{
            "value":"[parameters('GiantSwarmTags')]"
          },
          "vmssName":{
            "value":"[concat(parameters('clusterID'), '-worker')]"
          },
          "vmssMSIEnabled":{
            "value":"[parameters('vmssMSIEnabled')]"
          },
          "vmssVmSize":{
            "value":"[parameters('workerNodes')[0].vmSize]"
          },
          "vmssVmCount":{
            "value":"[length(parameters('workerNodes'))]"
          },
          "vmssStorageAccountType":{
            "value":"[if(contains(variables('vmssStandardLrsSize'), parameters('workerNodes')[0].vmSize), 'Standard_LRS', 'Premium_LRS')]"
          },
          "vmssVmDataDisks":{
            "value":[
              {
                "caching": "ReadWrite",
                "createOption": "Empty",
                "diskSizeGB": "[if(greater(parameters('workerNodes')[0].dockerVolumeSizeGB, 0), parameters('workerNodes')[0].dockerVolumeSizeGB, 50)]",
                "managedDisk":
                {
                  "storageAccountType": "[if(contains(variables('vmssStandardLrsSize'), parameters('workerNodes')[0].vmSize), 'Standard_LRS', 'Premium_LRS')]"
                },
                "lun": 21
              },
              {
                "caching": "ReadWrite",
                "createOption": "Empty",
                "diskSizeGB": "[if(greater(parameters('workerNodes')[0].kubeletVolumeSizeGB, 0), parameters('workerNodes')[0].kubeletVolumeSizeGB, 100)]",
                "managedDisk":
                {
                  "storageAccountType": "[if(contains(variables('vmssStandardLrsSize'), parameters('workerNodes')[0].vmSize), 'Standard_LRS', 'Premium_LRS')]"
                },
                "lun": 22
              }
            ]
          },
          "vmssSshUser":{
            "value":"[parameters('workerNodes')[0].adminUsername]"
          },
          "vmssSshPublicKey":{
            "value":"[parameters('workerNodes')[0].adminSSHKeyData]"
          },
          "vmssOsImagePublisher":{
            "value":"[parameters('workerNodes')[0].osImage.publisher]"
          },
          "vmssOsImageOffer":{
            "value":"[parameters('workerNodes')[0].osImage.offer]"
          },
          "vmssOsImageSKU":{
            "value":"[parameters('workerNodes')[0].osImage.sku]"
          },
          "vmssOsImageVersion":{
            "value":"[parameters('workerNodes')[0].osImage.version]"
          },
          "vmssOverprovision":{
            "value":"false"
          },
          "vmssVmCustomData":{
            "value":"[parameters('workerCloudConfigData')]"
          },
          "vmssLbBackendPools":{
            "value":[
              {
                "id":"[variables('kubernetesLBBackendID')]"
              }
            ]
          },
          "vmssVnetSubnetId":{
            "value":"[parameters('workerSubnetID')]"
          },
          "vmssUpgradePolicy":{
            "value":"Manual"
          },
          "zones":{
            "value":"[parameters('zones')]"
          }
        }
      }
    }
  ]
}`

func getARMTemplate() (map[string]interface{}, error) {
	contents := make(map[string]interface{})
	template := []byte(getARMTemplateAsString())
	if err := json.Unmarshal(template, &contents); err != nil {
		return nil, err
	}
	return contents, nil
}

func getARMTemplateAsString() string {
	return fmt.Sprintf(main, vmssTemplate)
}
