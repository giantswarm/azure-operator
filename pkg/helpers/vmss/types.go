package vmss

import (
	"github.com/giantswarm/certs/v4/pkg/certs"
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

func NewNode(distroVersion string, vmSize string, dockerVolumeSizeGB int, kubeletVolumeSizeGB int) Node {
	return Node{
		OSImage:             newNodeOSImageCoreOS(distroVersion),
		VMSize:              vmSize,
		DockerVolumeSizeGB:  dockerVolumeSizeGB,
		KubeletVolumeSizeGB: kubeletVolumeSizeGB,
	}
}

// newNodeOSImage provides OS information for Container Linux
func newNodeOSImageCoreOS(distroVersion string) NodeOSImage {
	return NodeOSImage{
		Offer:     "flatcar-container-linux-free",
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
