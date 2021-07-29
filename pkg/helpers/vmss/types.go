package vmss

import (
	"github.com/giantswarm/certs/v3/pkg/certs"
)

type Node struct {
	// OSImage is the vm OS image object.
	OSImage NodeOSImage `json:"osImage" yaml:"osImage"`
	// VMSize is the master vm size (e.g. Standard_A1)
	VMSize string `json:"vmSize" yaml:"vmSize"`
	// Size of the Disk mounted in /var/lib/docker
	DockerVolumeSizeGB int `json:"dockerVolumeSizeGB" yaml:"dockerVolumeSizeGB"`
	// Size of the Disk mounted in /var/lib/kubelet
	KubeletVolumeSizeGB int `json:"kubeletVolumeSizeGB" yaml:"kubeletVolumeSizeGB"`
}

// nodeOSImage provides OS information for Microsoft.Compute/virtualMachines
// templates. Official documentation for can be found here
// https://docs.microsoft.com/en-us/azure/templates/microsoft.compute/virtualmachines#ImageReference.
type NodeOSImage struct {
	// Offer is the image offered by the publisher (e.g. CoreOS).
	Offer string `json:"offer" yaml:"offer"`
	// Publisher is the image publisher (e.g GiantSwarm).
	Publisher string `json:"publisher" yaml:"publisher"`
	// SKU is the image SKU (e.g. Alpha).
	SKU string `json:"sku" yaml:"sku"`
	// Version is the image version (e.g. 1465.7.0).
	Version string `json:"version" yaml:"version"`
}

func NewNode(offer FlatcarOffer, distroVersion string, vmSize string, dockerVolumeSizeGB int, kubeletVolumeSizeGB int) Node {
	return Node{
		OSImage:             newNodeOSImageCoreOS(offer, distroVersion),
		VMSize:              vmSize,
		DockerVolumeSizeGB:  dockerVolumeSizeGB,
		KubeletVolumeSizeGB: kubeletVolumeSizeGB,
	}
}

type FlatcarOffer string

const (
	FlatcarFree FlatcarOffer = "flatcar-container-linux-free"
	FlatcarPro  FlatcarOffer = "flatcar_pro"
)

// newNodeOSImage provides OS information for Container Linux
func newNodeOSImageCoreOS(offer FlatcarOffer, distroVersion string) NodeOSImage {
	return NodeOSImage{
		Offer:     string(offer),
		Publisher: "kinvolk",
		SKU:       "stable",
		Version:   distroVersion,
	}
}

// SmallCloudconfigConfig represents the data structure required for executing
// the small cloudconfig template.
type SmallCloudconfigConfig struct {
	BlobURL       string
	CertsFiles    []certs.File
	EncryptionKey string
	InitialVector string
	InstanceRole  string
}
