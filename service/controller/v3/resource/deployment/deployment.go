package deployment

import (
	"context"

	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-02-01/resources"
	"github.com/Azure/go-autorest/autorest/to"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/service/controller/v3/key"
)

const (
	templateContentVersion = "1.0.0.0"
)

func getDeploymentNames() []string {
	return []string{
		mainDeploymentName,
	}
}

func (r Resource) newDeployment(ctx context.Context, customObject providerv1alpha1.AzureConfig, overwrites map[string]interface{}) (azureresource.Deployment, error) {
	var masterNodes []node
	for _, m := range customObject.Spec.Azure.Masters {
		n := node{
			AdminUsername:   key.AdminUsername(customObject),
			AdminSSHKeyData: key.AdminSSHKeyData(customObject),
			OSImage:         newNodeOSImageCoreOS_1745_7_0(),
			VMSize:          m.VMSize,
		}
		masterNodes = append(masterNodes, n)
	}

	var workerNodes []node
	for _, w := range customObject.Spec.Azure.Workers {
		n := node{
			AdminUsername:   key.AdminUsername(customObject),
			AdminSSHKeyData: key.AdminSSHKeyData(customObject),
			OSImage:         newNodeOSImageCoreOS_1745_7_0(),
			VMSize:          w.VMSize,
		}
		workerNodes = append(workerNodes, n)
	}

	masterCloudConfig, err := r.cloudConfig.NewMasterCloudConfig(customObject)
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	workerCloudConfig, err := r.cloudConfig.NewWorkerCloudConfig(customObject)
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	defaultParams := map[string]interface{}{
		"clusterID":                     key.ClusterID(customObject),
		"virtualNetworkName":            key.VnetName(customObject),
		"virtualNetworkCidr":            key.VnetCIDR(customObject),
		"calicoSubnetCidr":              key.VnetCalicoSubnetCIDR(customObject),
		"masterSubnetCidr":              key.VnetMasterSubnetCIDR(customObject),
		"workerSubnetCidr":              key.VnetWorkerSubnetCIDR(customObject),
		"masterNodes":                   masterNodes,
		"workerNodes":                   workerNodes,
		"dnsZones":                      key.DNSZones(customObject),
		"hostClusterCidr":               r.azure.HostCluster.CIDR,
		"hostClusterResourceGroupName":  r.azure.HostCluster.ResourceGroup,
		"hostClusterVirtualNetworkName": r.azure.HostCluster.VirtualNetwork,
		"kubernetesAPISecurePort":       key.APISecurePort(customObject),
		"masterCloudConfigData":         masterCloudConfig,
		"workerCloudConfigData":         workerCloudConfig,
		"vmssMSIEnabled":                r.azure.MSI.Enabled,
		"templatesBaseURI":              baseTemplateURI(r.templateVersion),
	}

	d := azureresource.Deployment{
		Properties: &azureresource.DeploymentProperties{
			Mode:       azureresource.Incremental,
			Parameters: key.ToParameters(defaultParams, overwrites),
			TemplateLink: &azureresource.TemplateLink{
				URI:            to.StringPtr(templateURI(r.templateVersion, mainTemplate)),
				ContentVersion: to.StringPtr(key.TemplateContentVersion),
			},
		},
	}

	return d, nil
}
