package deployment

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/giantswarm/microerror"
)

func getDeploymentTemplateChecksum(deployment resources.Deployment) (string, error) {
	templateLink := deployment.Properties.TemplateLink
	if templateLink == nil {
		return "", microerror.Mask(nilTemplateLinkError)
	}

	resp, err := http.Get(*templateLink.URI)
	if err != nil {
		return "", microerror.Mask(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", microerror.Mask(unableToGetTemplateError)
	}

	digest := sha256.New()
	_, err = io.Copy(digest, resp.Body)
	if err != nil {
		return "", microerror.Mask(err)
	}

	hash := fmt.Sprintf("%x", digest.Sum(nil))

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
