package template

import (
	"encoding/json"
	"reflect"

	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/giantswarm/microerror"
	"github.com/markbates/pkger"
)

// GetARMTemplate returns the ARM template reading a json file locally using pkger.
func GetARMTemplate() (map[string]interface{}, error) {
	contents := make(map[string]interface{})

	f, err := pkger.Open("/service/controller/resource/nodepool/template/main.json")
	if err != nil {
		return contents, microerror.Mask(err)
	}
	defer f.Close()

	d := json.NewDecoder(f)
	if err := d.Decode(&contents); err != nil {
		return contents, microerror.Mask(err)
	}
	return contents, microerror.Mask(err)
}

func NewDeployment(templateParams Parameters) (azureresource.Deployment, error) {
	armTemplate, err := GetARMTemplate()
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	return azureresource.Deployment{
		Properties: &azureresource.DeploymentProperties{
			Mode:       azureresource.Incremental,
			Parameters: templateParams.ToDeployParams(),
			Template:   armTemplate,
		},
	}, nil
}

func Diff(currentDeployment azureresource.DeploymentExtended, desiredDeployment azureresource.Deployment) ([]string, error) {
	var changes []string

	currentParameters, err := NewFromExtendedDeployment(currentDeployment)
	if err != nil {
		return changes, microerror.Mask(err)
	}

	desiredParameters, err := NewFromDeployment(desiredDeployment)
	if err != nil {
		return changes, microerror.Mask(err)
	}

	// If any of the following fields change, it means the deployments are not in sync.
	// We are not taking the field `VMCustomData` in consideration because it comes empty from Azure.
	// That's ok because changing `VMCustomData` would mean changing the `AzureOperatorVersion` field.
	if currentParameters.AzureOperatorVersion != desiredParameters.AzureOperatorVersion {
		changes = append(changes, "azureOperatorVersion")
	}
	if currentParameters.ClusterID != desiredParameters.ClusterID {
		changes = append(changes, "clusterID")
	}
	if currentParameters.NodepoolName != desiredParameters.NodepoolName {
		changes = append(changes, "nodepoolName")
	}
	if currentParameters.SubnetName != desiredParameters.SubnetName {
		changes = append(changes, "subnetName")
	}
	if currentParameters.VMSize != desiredParameters.VMSize {
		changes = append(changes, "vmSize")
	}
	if currentParameters.VnetName != desiredParameters.VnetName {
		changes = append(changes, "vnetName")
	}
	if !reflect.DeepEqual(currentParameters.DataDisks, desiredParameters.DataDisks) {
		changes = append(changes, "dataDisks")
	}
	if !reflect.DeepEqual(currentParameters.Scaling, desiredParameters.Scaling) {
		changes = append(changes, "scaling")
	}
	if !reflect.DeepEqual(currentParameters.OSImage, desiredParameters.OSImage) {
		changes = append(changes, "osImage")
	}
	if !reflect.DeepEqual(currentParameters.Zones, desiredParameters.Zones) {
		changes = append(changes, "zones")
	}

	return changes, nil
}
