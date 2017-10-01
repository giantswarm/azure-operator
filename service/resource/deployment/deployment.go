package deployment

import (
	"github.com/giantswarm/certificatetpr"

	"github.com/giantswarm/azuretpr"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/service/key"
)

const (
	mainDeploymentName     = "cluster-main-template"
	templateContentVersion = "1.0.0.0"
)

func getDeploymentNames() []string {
	return []string{
		mainDeploymentName,
	}
}

func (r Resource) newMainDeployment(cluster azuretpr.CustomObject) (Deployment, error) {
	certs, err := r.certWatcher.SearchCerts(key.ClusterID(cluster))
	if err != nil {
		return Deployment{}, microerror.Mask(err)
	}

	// Convert certs files into a collection of key vault secrets.
	certSecrets := convertCertsToSecrets(certs)

	masterCloudConfig, err := r.cloudConfig.NewMasterCloudConfig(cluster)
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
		"mastersCustomConfig":           cluster.Spec.Azure.Masters,
		"kubernetesAPISecurePort":       cluster.Spec.Cluster.Kubernetes.API.SecurePort,
		"etcdPort":                      cluster.Spec.Cluster.Etcd.Port,
		"kubernetesIngressSecurePort":   cluster.Spec.Cluster.Kubernetes.IngressController.SecurePort,
		"kubernetesIngressInsecurePort": cluster.Spec.Cluster.Kubernetes.IngressController.InsecurePort,
		"masterCloudConfigData":         masterCloudConfig,
		"keyVaultName":                  key.KeyVaultName(cluster),
		"keyVaultSecretsObject":         certSecrets,
		"templatesBaseURI":              baseTemplateURI(r.templateVersion),
	}

	deployment := Deployment{
		Name:                   mainDeploymentName,
		Parameters:             convertParameters(params),
		ResourceGroup:          key.ClusterID(cluster),
		TemplateURI:            templateURI(r.templateVersion, mainTemplate),
		TemplateContentVersion: templateContentVersion,
	}

	return deployment, nil
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
			SecretName:  key.SecretName(asset.Component, asset.Type),
			SecretValue: string(value),
		}

		secretsList = append(secretsList, secret)
	}

	return keyVaultSecrets{
		Secrets: secretsList,
	}
}
