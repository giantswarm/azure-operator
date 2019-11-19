package instance

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/giantswarm/azure-operator/service/controller/v12/key"
	"io/ioutil"
	"net/http"
)

func getDeploymentTemplateChecksum(deployment resources.Deployment) (string, error) {

	templateLink := deployment.Properties.TemplateLink
	if templateLink == nil {
		return "", nilTemplateLiknError
	}

	resp, err := http.Get(*templateLink.URI)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	hash := fmt.Sprintf("%x", sha256.Sum256(body))

	return hash, nil
}

func getDeploymentParametersChecksum(deployment resources.Deployment) (string, error) {
	// I can't use deployment.Properties.Parameters because it contains some dynamic data that changes at every loop
	// (at least the 'masterCloudConfigData' and 'workerCloudConfigData' but possibly other fields as well).
	// Currently I'm using just the ignition template to calculate the checksum.
	// This is error prone and should be fixed somehow.
	// todo read above
	jsonStr, err := json.Marshal(key.CloudConfigSmallTemplates())
	if err != nil {
		return "", err
	}

	hash := fmt.Sprintf("%x", sha256.Sum256(jsonStr))

	return hash, nil
}
