package instance

import (
	"context"

	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest/to"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/service/controller/v4patch2/controllercontext"
	"github.com/giantswarm/azure-operator/service/controller/v4patch2/key"
)

func (r Resource) newDeployment(ctx context.Context, obj providerv1alpha1.AzureConfig, overwrites map[string]interface{}) (azureresource.Deployment, error) {
	var masterNodes []node
	for _, m := range obj.Spec.Azure.Masters {
		n := node{
			AdminUsername:   key.AdminUsername(obj),
			AdminSSHKeyData: key.AdminSSHKeyData(obj),
			OSImage:         newNodeOSImageCoreOS_1967_5_0(),
			VMSize:          m.VMSize,
		}
		masterNodes = append(masterNodes, n)
	}

	var workerNodes []node
	for _, w := range obj.Spec.Azure.Workers {
		n := node{
			AdminUsername:   key.AdminUsername(obj),
			AdminSSHKeyData: key.AdminSSHKeyData(obj),
			OSImage:         newNodeOSImageCoreOS_1967_5_0(),
			VMSize:          w.VMSize,
		}
		workerNodes = append(workerNodes, n)
	}

	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}
	err = cc.Validate()
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	masterCloudConfig, err := cc.CloudConfig.NewMasterCloudConfig(obj)
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	workerCloudConfig, err := cc.CloudConfig.NewWorkerCloudConfig(obj)
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	defaultParams := map[string]interface{}{
		"apiLBBackendPoolID":    cc.APILBBackendPoolID,
		"clusterID":             key.ClusterID(obj),
		"etcdLBBackendPoolID":   cc.EtcdLBBackendPoolID,
		"masterCloudConfigData": masterCloudConfig,
		"masterNodes":           masterNodes,
		"masterSubnetID":        cc.MasterSubnetID,
		"templatesBaseURI":      baseTemplateURI(r.templateVersion),
		"vmssMSIEnabled":        r.azure.MSI.Enabled,
		"workerCloudConfigData": workerCloudConfig,
		"workerNodes":           workerNodes,
		"workerSubnetID":        cc.WorkerSubnetID,
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
