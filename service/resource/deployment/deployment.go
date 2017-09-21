package deployment

import (
	"fmt"

	"github.com/giantswarm/certificatetpr"

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

	certs, err := r.certWatcher.SearchCerts(key.ClusterID(cluster))
	if err != nil {
		return Deployment{}, microerror.Mask(err)
	}

	// Convert certs files into a collection of key vault secrets.
	certSecrets := convertCertsToSecrets(certs)

	// TODO Master CloudConfig will be passed in as a template parameter.
	_, err = r.cloudConfig.NewMasterCloudConfig(cluster)
	if err != nil {
		return Deployment{}, microerror.Mask(err)
	}

	// TODO Worker CloudConfig will be passed in as a template parameter.
	_, err = r.cloudConfig.NewWorkerCloudConfig(cluster)
	if err != nil {
		return Deployment{}, microerror.Mask(err)
	}

	params := map[string]interface{}{
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
		"keyVaultName":                  key.KeyVaultName(cluster),
		"keyVaultSecretsObject":         certSecrets,
		"templatesBaseURI":              templateBaseURI,
	}

	deployment := Deployment{
		Name:            mainDeploymentName,
		Parameters:      convertParameters(params),
		ResourceGroup:   key.ClusterID(cluster),
		TemplateURI:     mainTemplateURI,
		TemplateVersion: templateVersion,
	}

	return deployment, nil
}

func (r Resource) templateBaseURI() (string, error) {
	if r.uriVersion == "" {
		return "", microerror.Maskf(invalidConfigError, "Missing URI version for ARM templates")
	}

	return fmt.Sprintf(templatesBaseURI, r.uriVersion), nil
}

func (r Resource) templateURI(templateFileName string) (string, error) {
	baseURI, err := r.templateBaseURI()
	if err != nil {
		return "", microerror.Mask(err)
	}

	return fmt.Sprintf("%s%s", baseURI, templateFileName), nil
}

// convertParameters converts the map into the structure used by the Azure API.
func convertParameters(inputs map[string]interface{}) map[string]interface{} {
	params := make(map[string]interface{}, len(inputs))
	for key, val := range inputs {
		params[key] = struct {
			Value interface{}
		}{
			Value: val,
		}
	}

	return params
}

// convertCertsToSecrets converts the certificate assets to a keyVaultSecrets
// collection so it can be passed as a secure object template parameter.
func convertCertsToSecrets(certs certificatetpr.AssetsBundle) keyVaultSecrets {
	var secretsList []keyVaultSecret

	for asset, value := range certs {
		secret := keyVaultSecret{
			SecretName:  key.SecretName(asset),
			SecretValue: string(value),
		}

		secretsList = append(secretsList, secret)
	}

	return keyVaultSecrets{
		Secrets: secretsList,
	}
}
