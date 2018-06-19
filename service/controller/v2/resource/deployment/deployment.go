package deployment

import (
	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-02-01/resources"
	"github.com/Azure/go-autorest/autorest/to"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/service/controller/v2/key"
)

const (
	templateContentVersion = "1.0.0.0"
)

func getDeploymentNames() []string {
	return []string{
		mainDeploymentName,
	}
}

func (r Resource) newDeployment(obj providerv1alpha1.AzureConfig, overwrites map[string]interface{}) (azureresource.Deployment, error) {
	var masterNodes []node
	for _, m := range obj.Spec.Azure.Masters {
		n := node{
			AdminUsername:   key.AdminUsername(obj),
			AdminSSHKeyData: key.AdminSSHKeyData(obj),
			OSImage:         newNodeOSImageCoreOS_1688_5_3(),
			VMSize:          m.VMSize,
		}
		masterNodes = append(masterNodes, n)
	}

	var workerNodes []node
	for _, w := range obj.Spec.Azure.Workers {
		n := node{
			AdminUsername:   key.AdminUsername(obj),
			AdminSSHKeyData: key.AdminSSHKeyData(obj),
			OSImage:         newNodeOSImageCoreOS_1688_5_3(),
			VMSize:          w.VMSize,
		}
		workerNodes = append(workerNodes, n)
	}

	masterCloudConfig, err := r.cloudConfig.NewMasterCloudConfig(obj)
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	workerCloudConfig, err := r.cloudConfig.NewWorkerCloudConfig(obj)
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	defaultParams := map[string]interface{}{
		"clusterID":                     key.ClusterID(obj),
		"virtualNetworkName":            key.VnetName(obj),
		"virtualNetworkCidr":            key.VnetCIDR(obj),
		"calicoSubnetCidr":              key.VnetCalicoSubnetCIDR(obj),
		"masterSubnetCidr":              key.VnetMasterSubnetCIDR(obj),
		"workerSubnetCidr":              key.VnetWorkerSubnetCIDR(obj),
		"masterNodes":                   masterNodes,
		"workerNodes":                   workerNodes,
		"dnsZones":                      obj.Spec.Azure.DNSZones,
		"hostClusterCidr":               r.azure.HostCluster.CIDR,
		"hostClusterResourceGroupName":  r.azure.HostCluster.ResourceGroup,
		"hostClusterVirtualNetworkName": r.azure.HostCluster.VirtualNetwork,
		"kubernetesAPISecurePort":       obj.Spec.Cluster.Kubernetes.API.SecurePort,
		"masterCloudConfigData":         masterCloudConfig,
		"workerCloudConfigData":         workerCloudConfig,
		"vmssMSIEnabled":                r.azure.MSI.Enabled,
		"templatesBaseURI":              baseTemplateURI(r.templateVersion),
	}

	d := azureresource.Deployment{
		Properties: &azureresource.DeploymentProperties{
			Mode:       azureresource.Incremental,
			Parameters: convertParameters(defaultParams, overwrites),
			TemplateLink: &azureresource.TemplateLink{
				URI:            to.StringPtr(templateURI(r.templateVersion, mainTemplate)),
				ContentVersion: to.StringPtr(templateContentVersion),
			},
		},
	}

	return d, nil
}

// convertParameters merges the input maps and converts the result into the
// structure used by the Azure API. Note that the order of inputs is relevant.
// Default parameters should be given first. Data of the following maps will
// overwrite eventual data of preceeding maps. This mechanism is used for e.g.
// setting the initialProvisioning parameter accordingly to the cluster's state.
func convertParameters(list ...map[string]interface{}) map[string]interface{} {
	allParams := map[string]interface{}{}

	for _, l := range list {
		for key, val := range l {
			allParams[key] = struct {
				Value interface{}
			}{
				Value: val,
			}
		}
	}

	return allParams
}
