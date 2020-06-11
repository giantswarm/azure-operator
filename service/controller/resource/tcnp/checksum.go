package tcnp

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/giantswarm/microerror"
	v1alpha32 "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
)

func getDeploymentTemplateChecksum(deployment resources.Deployment) (string, error) {
	template := deployment.Properties.Template.(map[string]interface{})
	jsonStr, err := json.Marshal(template)
	if err != nil {
		return "", microerror.Mask(err)
	}

	// Calculate the sha256 hash of the JSON.
	hash := fmt.Sprintf("%x", sha256.Sum256(jsonStr))

	return hash, nil
}

func getDeploymentParametersChecksum(deployment resources.Deployment) (string, error) {
	params := deployment.Properties.Parameters.(map[string]interface{})

	// Create a JSON with the whole adjusted parameters.
	jsonStr, err := json.Marshal(params)
	if err != nil {
		return "", microerror.Mask(err)
	}

	// Calculate the sha256 hash of the JSON.
	hash := fmt.Sprintf("%x", sha256.Sum256(jsonStr))

	return hash, nil
}

func (r *Resource) getCurrentDeploymentChecksums(ctx context.Context, customObject v1alpha32.AzureMachinePool) (string, string, error) {
	currentDeploymentTemplateChk, err := r.getResourceStatus(ctx, customObject, DeploymentTemplateChecksum)
	if err != nil {
		return "", "", microerror.Mask(err)
	}

	currentDeploymentParametersChk, err := r.getResourceStatus(ctx, customObject, DeploymentParametersChecksum)
	if err != nil {
		return "", "", microerror.Mask(err)
	}

	return currentDeploymentTemplateChk, currentDeploymentParametersChk, nil
}

func (r *Resource) getDesiredDeploymentChecksums(ctx context.Context, desiredDeployment resources.Deployment) (string, string, error) {
	desiredDeploymentTemplateChk, err := getDeploymentTemplateChecksum(desiredDeployment)
	if err != nil {
		return "", "", microerror.Mask(err)
	}

	desiredDeploymentParametersChk, err := getDeploymentParametersChecksum(desiredDeployment)
	if err != nil {
		return "", "", microerror.Mask(err)
	}

	return desiredDeploymentTemplateChk, desiredDeploymentParametersChk, nil
}
