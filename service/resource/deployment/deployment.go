package deployment

import (
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/certs"
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

func (r Resource) newMainDeployment(obj providerv1alpha1.AzureConfig) (Deployment, error) {
	certs, err := r.certsSearcher.SearchCluster(key.ClusterID(obj))
	if err != nil {
		return Deployment{}, microerror.Mask(err)
	}

	// Convert certs files into a collection of key vault secrets.
	certSecrets := convertCertsToSecrets(certs)

	var masterNodes []node
	for _, m := range obj.Spec.Azure.Masters {
		n := node{
			AdminUsername:   key.AdminUsername(obj),
			AdminSSHKeyData: key.AdminSSHKeyData(obj),
			OSImage:         newNodeOSImageCoreOS_1465_7_0(),
			VMSize:          m.VMSize,
		}
		masterNodes = append(masterNodes, n)
	}

	var workerNodes []node
	for _, w := range obj.Spec.Azure.Workers {
		n := node{
			AdminUsername:   key.AdminUsername(obj),
			AdminSSHKeyData: key.AdminSSHKeyData(obj),
			OSImage:         newNodeOSImageCoreOS_1465_7_0(),
			VMSize:          w.VMSize,
		}
		workerNodes = append(workerNodes, n)
	}

	masterCloudConfig, err := r.cloudConfig.NewMasterCloudConfig(obj)
	if err != nil {
		return Deployment{}, microerror.Mask(err)
	}

	workerCloudConfig, err := r.cloudConfig.NewWorkerCloudConfig(obj)
	if err != nil {
		return Deployment{}, microerror.Mask(err)
	}

	params := map[string]interface{}{
		"clusterID":                     key.ClusterID(obj),
		"virtualNetworkCidr":            obj.Spec.Azure.VirtualNetwork.CIDR,
		"calicoSubnetCidr":              key.CalicoSubnetCidr(obj),
		"masterSubnetCidr":              obj.Spec.Azure.VirtualNetwork.MasterSubnetCIDR,
		"workerSubnetCidr":              obj.Spec.Azure.VirtualNetwork.WorkerSubnetCIDR,
		"masterNodes":                   masterNodes,
		"workerNodes":                   workerNodes,
		"dnsZones":                      obj.Spec.Azure.DNSZones,
		"hostClusterCidr":               obj.Spec.Azure.HostCluster.CIDR,
		"kubernetesAPISecurePort":       obj.Spec.Cluster.Kubernetes.API.SecurePort,
		"kubernetesIngressSecurePort":   obj.Spec.Cluster.Kubernetes.IngressController.SecurePort,
		"kubernetesIngressInsecurePort": obj.Spec.Cluster.Kubernetes.IngressController.InsecurePort,
		"masterCloudConfigData":         masterCloudConfig,
		"workerCloudConfigData":         workerCloudConfig,
		"keyVaultName":                  key.KeyVaultName(obj),
		"keyVaultSecretsObject":         certSecrets,
		"templatesBaseURI":              baseTemplateURI(r.templateVersion),
	}

	deployment := Deployment{
		Name:                   mainDeploymentName,
		Parameters:             convertParameters(params),
		ResourceGroup:          key.ClusterID(obj),
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
func convertCertsToSecrets(certificates certs.Cluster) keyVaultSecrets {
	var secrets []keyVaultSecret

	for _, f := range certs.NewFilesCluster(certificates) {
		s := keyVaultSecret{
			SecretName:  key.KeyVaultKey(f.AbsolutePath),
			SecretValue: string(f.Data),
		}
		secrets = append(secrets, s)
	}

	return keyVaultSecrets{
		Secrets: secrets,
	}
}
