package instance

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

func (r Resource) newDeployment(obj providerv1alpha1.AzureConfig, overwrites map[string]interface{}) (azureresource.Deployment, error) {
	var masterNodes []node
	for _, m := range obj.Spec.Azure.Masters {
		n := node{
			AdminUsername:   key.AdminUsername(obj),
			AdminSSHKeyData: key.AdminSSHKeyData(obj),
			OSImage:         newNodeOSImageCoreOS_1745_7_0(),
			VMSize:          m.VMSize,
		}
		masterNodes = append(masterNodes, n)
	}

	var workerNodes []node
	for _, w := range obj.Spec.Azure.Workers {
		n := node{
			AdminUsername:   key.AdminUsername(obj),
			AdminSSHKeyData: key.AdminSSHKeyData(obj),
			OSImage:         newNodeOSImageCoreOS_1745_7_0(),
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
		"clusterID":                   key.ClusterID(obj),
		"masterCloudConfigData":       masterCloudConfig,
		"masterNodes":                 masterNodes,
		"masterVersionBundleVersions": TODO,
		"templatesBaseURI":            baseTemplateURI(r.templateVersion),
		"vmssMSIEnabled":              r.azure.MSI.Enabled,
		"workerCloudConfigData":       workerCloudConfig,
		"workerNodes":                 workerNodes,
		"workerVersionBundleVersions": TODO,
	}

	d := azureresource.Deployment{
		Properties: &azureresource.DeploymentProperties{
			Mode:       azureresource.Incremental,
			Parameters: key.ToParameters(defaultParams, overwrites),
			TemplateLink: &azureresource.TemplateLink{
				URI:            to.StringPtr(templateURI(r.templateVersion, mainTemplate)),
				ContentVersion: to.StringPtr(templateContentVersion),
			},
		},
	}

	return d, nil
}
