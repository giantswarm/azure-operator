package deployment

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/giantswarm/microerror"
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
