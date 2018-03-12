package deployment

import (
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/service/azureconfig/v1/key"
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

func (r Resource) newMainDeployment(obj providerv1alpha1.AzureConfig) (deployment, error) {
	_, err := r.certsSearcher.SearchCluster(key.ClusterID(obj))
	if err != nil {
		return deployment{}, microerror.Mask(err)
	}

	var masterNodes []node
	for _, m := range obj.Spec.Azure.Masters {
		n := node{
			AdminUsername:   key.AdminUsername(obj),
			AdminSSHKeyData: key.AdminSSHKeyData(obj),
			OSImage:         newNodeOSImageCoreOS_1632_3_0(),
			VMSize:          m.VMSize,
		}
		masterNodes = append(masterNodes, n)
	}

	var workerNodes []node
	for _, w := range obj.Spec.Azure.Workers {
		n := node{
			AdminUsername:   key.AdminUsername(obj),
			AdminSSHKeyData: key.AdminSSHKeyData(obj),
			OSImage:         newNodeOSImageCoreOS_1632_3_0(),
			VMSize:          w.VMSize,
		}
		workerNodes = append(workerNodes, n)
	}

	masterCloudConfig, err := r.cloudConfig.NewMasterCloudConfig(obj)
	if err != nil {
		return deployment{}, microerror.Mask(err)
	}

	workerCloudConfig, err := r.cloudConfig.NewWorkerCloudConfig(obj)
	if err != nil {
		return deployment{}, microerror.Mask(err)
	}

	params := map[string]interface{}{
		"clusterID":                     key.ClusterID(obj),
		"virtualNetworkCidr":            key.VnetCIDR(obj),
		"calicoSubnetCidr":              key.VnetCalicoSubnetCIDR(obj),
		"masterSubnetCidr":              key.VnetMasterSubnetCIDR(obj),
		"workerSubnetCidr":              key.VnetWorkerSubnetCIDR(obj),
		"masterNodes":                   masterNodes,
		"workerNodes":                   workerNodes,
		"dnsZones":                      obj.Spec.Azure.DNSZones,
		"hostClusterCidr":               obj.Spec.Azure.HostCluster.CIDR,
		"kubernetesAPISecurePort":       obj.Spec.Cluster.Kubernetes.API.SecurePort,
		"kubernetesIngressSecurePort":   obj.Spec.Cluster.Kubernetes.IngressController.SecurePort,
		"kubernetesIngressInsecurePort": obj.Spec.Cluster.Kubernetes.IngressController.InsecurePort,
		"masterCloudConfigData":         masterCloudConfig,
		"workerCloudConfigData":         workerCloudConfig,
		"templatesBaseURI":              baseTemplateURI(r.templateVersion),
	}

	d := deployment{
		Name:                   mainDeploymentName,
		Parameters:             convertParameters(params),
		ResourceGroup:          key.ClusterID(obj),
		TemplateURI:            templateURI(r.templateVersion, mainTemplate),
		TemplateContentVersion: templateContentVersion,
	}

	return d, nil
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
