package deployment

import (
	"fmt"

	"github.com/giantswarm/azuretpr"

	"github.com/giantswarm/azure-operator/service/key"
)

const (
	mainDeploymentName          = "cluster-main-template"
	templateBaseURI             = "https://raw.githubusercontent.com/giantswarm/azure-operator/main-arm-template/service/arm_templates"
	templateVersion             = "1.0.0.0"
	mainTemplate                = "main.json"
	clusterSetupTemplate        = "cluster_setup.json"
	securityGroupsSetupTemplate = "security_groups_setup.json"
	virtualNetworkSetupTemplate = "virtual_network_setup.json"
)

func getDeploymentNames() []string {
	return []string{
		mainDeploymentName,
	}
}

func newMainDeployment(cluster azuretpr.CustomObject) (Deployment, error) {
	deployment := Deployment{
		Name: mainDeploymentName,
		Parameters: map[string]interface{}{
			"clusterID":                     key.ClusterID(cluster),
			"storageAccountType":            "Standard_LRS", // TODO Move to azuretpr
			"virtualNetworkCidr":            cluster.Spec.Azure.VirtualNetwork.CIDR,
			"masterSubnetCidr":              cluster.Spec.Azure.VirtualNetwork.MasterSubnetCIDR,
			"workerSubnetCidr":              cluster.Spec.Azure.VirtualNetwork.WorkerSubnetCIDR,
			"apiLoadBalancerCidr":           "10.1.1.0/25", // TODO Remove once LB resources are created,
			"etcdLoadBalancerCidr":          "10.1.1.128/25",
			"ingressLoadBalancerCidr":       "10.1.2.0/25",
			"kubernetesAPISecurePort":       fmt.Sprintf("%d", cluster.Spec.Cluster.Kubernetes.API.SecurePort),
			"etcdPort":                      fmt.Sprintf("%d", cluster.Spec.Cluster.Etcd.Port),
			"kubernetesIngressSecurePort":   fmt.Sprintf("%d", cluster.Spec.Cluster.Kubernetes.IngressController.SecurePort),
			"kubernetesIngressInsecurePort": fmt.Sprintf("%d", cluster.Spec.Cluster.Kubernetes.IngressController.InsecurePort),
			"clusterSetupTemplate":          templateURI(clusterSetupTemplate),
			"securityGroupsSetupTemplate":   templateURI(securityGroupsSetupTemplate),
			"virtualNetworkSetupTemplate":   templateURI(virtualNetworkSetupTemplate),
		},
		ResourceGroup:   key.ClusterID(cluster),
		TemplateURI:     templateURI(mainTemplate),
		TemplateVersion: templateVersion,
	}

	return deployment, nil
}

func templateURI(templateFileName string) string {
	return fmt.Sprintf("%s/%s", templateBaseURI, templateFileName)
}
