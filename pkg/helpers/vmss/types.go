package vmss

import (
	"github.com/giantswarm/certs"

	"github.com/giantswarm/azure-operator/v3/service/controller/key"
)

type Node struct {
	// AdminUsername is the vm administrator username
	AdminUsername string `json:"adminUsername" yaml:"adminUsername"`
	//  AdminSSHKeyData is the vm administrator ssh public key
	AdminSSHKeyData string `json:"adminSSHKeyData" yaml:"adminSSHKeyData"`
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

func NewNode(adminUsername string, adminSSHKeyData string, vmSize string, dockerVolumeSizeGB int, kubeletVolumeSizeGB int) Node {
	return Node{
		AdminUsername:       adminUsername,
		AdminSSHKeyData:     adminSSHKeyData,
		OSImage:             newNodeOSImageCoreOS(),
		VMSize:              vmSize,
		DockerVolumeSizeGB:  dockerVolumeSizeGB,
		KubeletVolumeSizeGB: kubeletVolumeSizeGB,
	}
}

// newNodeOSImage provides OS information for Container Linux
func newNodeOSImageCoreOS() NodeOSImage {
	return NodeOSImage{
		Offer:     "CoreOS",
		Publisher: "CoreOS",
		SKU:       "Stable",
		Version:   key.CoreosVersion,
	}
}

// SmallCloudconfigConfig represents the data structure required for executing
// the small cloudconfig template.
type SmallCloudconfigConfig struct {
	BlobURL       string
	CertsFiles    certs.Files
	EncryptionKey string
	InitialVector string
	InstanceRole  string
}
