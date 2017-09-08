package deployment

import (
	"fmt"

	"github.com/giantswarm/azuretpr"

	"github.com/giantswarm/azure-operator/service/key"
)

const (
	clusterIDParameterName     = "clusterID"
	clusterSetupDeploymentName = "cluster-setup"
	clusterSetupTemplate       = "cluster_setup.json"
	templateBaseURI            = "https://raw.githubusercontent.com/giantswarm/azure-operator/master/service/arm_templates"
	templateVersion            = "1.0.0.0"
)

func getDeploymentNames() []string {
	return []string{
		clusterSetupDeploymentName,
	}
}

func newClusterSetupDeployment(cluster azuretpr.CustomObject) (Deployment, error) {
	deployment := Deployment{
		Name: clusterSetupDeploymentName,
		Parameters: map[string]interface{}{
			clusterIDParameterName: key.ClusterID(cluster),
		},
		ResourceGroup:   key.ClusterID(cluster),
		TemplateURI:     templateURI(clusterSetupTemplate),
		TemplateVersion: templateVersion,
	}

	return deployment, nil
}

func templateURI(templateFileName string) string {
	return fmt.Sprintf("%s/%s", templateBaseURI, templateFileName)
}
