package deployment

const (
	clusterSetupDeploymentName = "cluster-setup"
)

func getDeploymentNames() []string {
	return []string{
		clusterSetupDeploymentName,
	}
}
