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

// IsOutOfDate returns whether or not two Azure ARM Deployments are equal.
func IsOutOfDate(currentDeployment azureresource.DeploymentExtended, desiredDeployment azureresource.Deployment) (bool, error) {
	if currentDeployment.IsHTTPStatus(404) {
		return true, nil
	}

	currentParameters, err := NewFromExtendedDeployment(currentDeployment)
	if err != nil {
		return false, microerror.Mask(err)
	}

	desiredParameters, err := NewFromDeployment(desiredDeployment)
	if err != nil {
		return false, microerror.Mask(err)
	}

	return !reflect.DeepEqual(currentParameters, desiredParameters), nil
}

// NeedToRolloutNodes tells whether or not we need to replace the existing VMs.
// We only don't roll out nodes when we change the scaling parameters.
func NeedToRolloutNodes(currentDeployment azureresource.DeploymentExtended, desiredDeployment azureresource.Deployment) (bool, error) {
	currentParameters, err := NewFromExtendedDeployment(currentDeployment)
	if err != nil {
		return false, microerror.Mask(err)
	}

	desiredParameters, err := NewFromDeployment(desiredDeployment)
	if err != nil {
		return false, microerror.Mask(err)
	}

	rolloutnodes := currentParameters.AzureOperatorVersion != desiredParameters.AzureOperatorVersion ||
		currentParameters.ClusterID != desiredParameters.ClusterID ||
		currentParameters.NodepoolName != desiredParameters.NodepoolName ||
		currentParameters.SSHPublicKey != desiredParameters.SSHPublicKey ||
		currentParameters.SubnetName != desiredParameters.SubnetName ||
		currentParameters.VMCustomData != desiredParameters.VMCustomData ||
		currentParameters.VMSize != desiredParameters.VMSize ||
		currentParameters.VnetName != desiredParameters.VnetName ||
		!reflect.DeepEqual(currentParameters.DataDisks, currentParameters.DataDisks) ||
		!reflect.DeepEqual(currentParameters.OSImage, currentParameters.OSImage) ||
		!reflect.DeepEqual(currentParameters.Zones, currentParameters.Zones)

	return rolloutnodes, nil
}
