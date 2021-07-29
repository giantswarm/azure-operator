package vmss

import (
	"encoding/base64"

	providerv1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v5/pkg/annotation"
	"github.com/giantswarm/azure-operator/v5/service/controller/key"
	"github.com/giantswarm/azure-operator/v5/service/controller/templates"
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

func GetMasterNodesConfiguration(obj providerv1alpha1.AzureConfig, distroVersion string) []Node {
	offer := getFlatcarOffer(obj)
	return getNodesConfiguration(offer, distroVersion, obj.Spec.Azure.Masters)
}

func GetWorkerNodesConfiguration(obj providerv1alpha1.AzureConfig, distroVersion string) []Node {
	offer := getFlatcarOffer(obj)
	return getNodesConfiguration(offer, distroVersion, obj.Spec.Azure.Workers)
}

func getNodesConfiguration(offer FlatcarOffer, distroVersion string, nodesSpecs []providerv1alpha1.AzureConfigSpecAzureNode) []Node {
	var nodes []Node
	for _, m := range nodesSpecs {
		n := NewNode(offer, distroVersion, m.VMSize, m.DockerVolumeSizeGB, m.KubeletVolumeSizeGB)
		nodes = append(nodes, n)
	}
	return nodes
}

func getFlatcarOffer(obj providerv1alpha1.AzureConfig) FlatcarOffer {
	offer := FlatcarFree
	if obj.Annotations[annotation.FlatcarPro] == "true" {
		offer = FlatcarPro
	}

	return offer
}
