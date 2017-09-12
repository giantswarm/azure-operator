package deployment

import (
	"fmt"

	"github.com/giantswarm/azuretpr"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/service/key"
)

const (
	mainDeploymentName          = "cluster-main-template"
	templatesBaseURI            = "https://raw.githubusercontent.com/giantswarm/azure-operator/%s/service/arm_templates/"
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

func (r Resource) newMainDeployment(cluster azuretpr.CustomObject) (Deployment, error) {
	templateBaseURI, err := r.templateBaseURI()
	if err != nil {
		return Deployment{}, microerror.Mask(err)
	}
	mainTemplateURI, err := r.templateURI(mainTemplate)
	if err != nil {
		return Deployment{}, microerror.Mask(err)
	}

	deployment := Deployment{
		Name: mainDeploymentName,
		Parameters: map[string]interface{}{
			"clusterID":                     key.ClusterID(cluster),
			"storageAccountType":            cluster.Spec.Azure.Storage.AccountType,
			"virtualNetworkCidr":            cluster.Spec.Azure.VirtualNetwork.CIDR,
			"masterSubnetCidr":              cluster.Spec.Azure.VirtualNetwork.MasterSubnetCIDR,
			"workerSubnetCidr":              cluster.Spec.Azure.VirtualNetwork.WorkerSubnetCIDR,
			"apiLoadBalancerCidr":           cluster.Spec.Azure.VirtualNetwork.LoadBalancer.APICIDR,
			"etcdLoadBalancerCidr":          cluster.Spec.Azure.VirtualNetwork.LoadBalancer.EtcdCIDR,
			"ingressLoadBalancerCidr":       cluster.Spec.Azure.VirtualNetwork.LoadBalancer.IngressCIDR,
			"kubernetesAPISecurePort":       cluster.Spec.Cluster.Kubernetes.API.SecurePort,
			"etcdPort":                      cluster.Spec.Cluster.Etcd.Port,
			"kubernetesIngressSecurePort":   cluster.Spec.Cluster.Kubernetes.IngressController.SecurePort,
			"kubernetesIngressInsecurePort": cluster.Spec.Cluster.Kubernetes.IngressController.InsecurePort,
			"templatesBaseURI":              templateBaseURI,
		},
		ResourceGroup:   key.ClusterID(cluster),
		TemplateURI:     mainTemplateURI,
		TemplateVersion: templateVersion,
	}

	return deployment, nil
}

func (r Resource) templateBaseURI() (string, error) {
	if r.armTemplateBranch == "" {
		return "", microerror.Maskf(invalidConfigError, "Missing ARM template branch")
	}

	return fmt.Sprintf(templatesBaseURI, r.armTemplateBranch), nil
}

func (r Resource) templateURI(templateFileName string) (string, error) {
	baseURI, err := r.templateBaseURI()
	if err != nil {
		return "", microerror.Mask(err)
	}

	return fmt.Sprintf("%s%s", baseURI, templateFileName), nil
}
