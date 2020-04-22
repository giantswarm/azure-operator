package helpers

import (
	"encoding/base64"

	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/service/controller/key"
	"github.com/giantswarm/azure-operator/service/controller/templates"
)

func RenderCloudConfig(blobURL string, encryptionKey string, initialVector string, instanceRole string) (string, error) {
	smallCloudconfigConfig := SmallCloudconfigConfig{
		BlobURL:       blobURL,
		EncryptionKey: encryptionKey,
		InitialVector: initialVector,
		InstanceRole:  instanceRole,
	}
	cloudConfig, err := templates.Render(key.CloudConfigSmallTemplates(), smallCloudconfigConfig)
	if err != nil {
		return "", microerror.Mask(err)
	}

	return base64.StdEncoding.EncodeToString([]byte(cloudConfig)), nil
}

func GetMasterNodesConfiguration(obj providerv1alpha1.AzureConfig) []Node {
	return getNodesConfiguration(key.AdminUsername(obj), key.AdminSSHKeyData(obj), obj.Spec.Azure.Masters)
}

func GetWorkerNodesConfiguration(obj providerv1alpha1.AzureConfig) []Node {
	return getNodesConfiguration(key.AdminUsername(obj), key.AdminSSHKeyData(obj), obj.Spec.Azure.Workers)
}

func getNodesConfiguration(adminUsername string, adminSSHKeyData string, nodesSpecs []providerv1alpha1.AzureConfigSpecAzureNode) []Node {
	var nodes []Node
	for _, m := range nodesSpecs {
		n := NewNode(adminUsername, adminSSHKeyData, m.VMSize, m.DockerVolumeSizeGB, m.KubeletVolumeSizeGB)
		nodes = append(nodes, n)
	}
	return nodes
}
