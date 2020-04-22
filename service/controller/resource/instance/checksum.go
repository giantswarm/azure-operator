package instance

import (
	"crypto/sha256"
	"encoding/base64"
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

	filteredParams := map[string]interface{}{}

	// Some of the fields in the Parameters property change at every iteration,
	// I have to filter out the changing data in order to be able to calculate a replicatable checksum.
	for k, v := range params {
		switch k {
		case "workerCloudConfigData":
			fallthrough
		case "masterCloudConfigData":
			// These two fields are base64 encoded, so first decode 'em.
			decoded, err := base64.StdEncoding.DecodeString(v.(struct{ Value interface{} }).Value.(string))
			if err != nil {
				return "", microerror.Mask(err)
			}

			// The decoded content is a JSON, convert it into a map to manipulate it.
			var m map[string]interface{}
			err = json.Unmarshal(decoded, &m)
			if err != nil {
				return "", microerror.Mask(err)
			}

			// Safely delete the [ignition][config] field which is the one that changes at every loop.
			// It is not semantically important and it can be ignored for the sake of checksum calculation.
			_, ok := m["ignition"]
			if ok {
				m["ignition"].(map[string]interface{})["config"] = nil
			}

			// Convert the modified map back to JSON.
			jsonStr, err := json.Marshal(m)
			if err != nil {
				return "", microerror.Mask(err)
			}

			// Encode back the JSON into base64.
			encoded := base64.StdEncoding.EncodeToString(jsonStr)

			// Add the new parameter into the filteredParameters.
			filteredParams[k] = encoded

			continue
		default:
			// All other fields are kept as-is.
			filteredParams[k] = v
		}
	}

	// Create a JSON with the whole adjusted parameters.
	jsonStr, err := json.Marshal(filteredParams)
	if err != nil {
		return "", microerror.Mask(err)
	}

	// Calculate the sha256 hash of the JSON.
	hash := fmt.Sprintf("%x", sha256.Sum256(jsonStr))

	return hash, nil
}
