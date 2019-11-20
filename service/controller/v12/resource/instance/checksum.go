package instance

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"io/ioutil"
	"net/http"
)

func getDeploymentTemplateChecksum(deployment resources.Deployment) (string, error) {

	templateLink := deployment.Properties.TemplateLink
	if templateLink == nil {
		return "", nilTemplateLinkError
	}

	resp, err := http.Get(*templateLink.URI)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", unableToGetTemplateError
	}

	body, err := ioutil.ReadAll(resp.Body)

	hash := fmt.Sprintf("%x", sha256.Sum256(body))

	return hash, nil
}

func getDeploymentParametersChecksum(deployment resources.Deployment) (string, error) {
	params := deployment.Properties.Parameters.(map[string]interface{})

	filteredParams := map[string]interface{}{}

	// some of the fields in the Parameters property change at every iteration
	// I have to filter out the changing data in order to be able to calculate a replicatable checksum
	for k, v := range params {
		switch k {
		case "workerCloudConfigData":
			fallthrough
		case "masterCloudConfigData":
			// these two fields are base64 encoded, so first decode 'em
			decoded, err := base64.StdEncoding.DecodeString(v.(struct{ Value interface{} }).Value.(string))
			if err != nil {
				return "", err
			}

			// the decoded content is a json, convert it into a map to manipulate it
			var m map[string]interface{}
			err = json.Unmarshal(decoded, &m)
			if err != nil {
				return "", err
			}

			// delete the [ignition][config] field which is the one that changes at every loop.
			// It is not semantically important and it can be ignored for the sake of checksum calculation
			m["ignition"].(map[string]interface{})["config"] = nil

			// convert the modified map back to json
			jsonStr, err := json.Marshal(m)
			if err != nil {
				return "", err
			}

			// encode back the json into base64
			encoded := base64.StdEncoding.EncodeToString(jsonStr)

			// add the new parameter into the filteredParameters
			filteredParams[k] = encoded

			continue
		default:
			// all other fields are kept as-is
			filteredParams[k] = v
		}
	}

	// create a json with the whole adjusted parameters
	jsonStr, err := json.Marshal(filteredParams)
	if err != nil {
		return "", err
	}

	// calculate the sha256 hash of the json
	hash := fmt.Sprintf("%x", sha256.Sum256(jsonStr))

	return hash, nil
}
