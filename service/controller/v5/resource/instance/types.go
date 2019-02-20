package instance

type node struct {
	// AdminUsername is the vm administrator username
	AdminUsername string `json:"adminUsername" yaml:"adminUsername"`
	//  AdminSSHKeyData is the vm administrator ssh public key
	AdminSSHKeyData string `json:"adminSSHKeyData" yaml:"adminSSHKeyData"`
	// OSImage is the vm OS image object.
	OSImage nodeOSImage `json:"osImage" yaml:"osImage"`
	// VMSize is the master vm size (e.g. Standard_A1)
	VMSize string `json:"vmSize" yaml:"vmSize"`
}

// nodeOSImage provides OS information for Microsoft.Compute/virtualMachines
// templates. Official documentation for can be found here
// https://docs.microsoft.com/en-us/azure/templates/microsoft.compute/virtualmachines#ImageReference.
type nodeOSImage struct {
	// Offer is the image offered by the publisher (e.g. CoreOS).
	Offer string `json:"offer" yaml:"offer"`
	// Publisher is the image publisher (e.g GiantSwarm).
	Publisher string `json:"publisher" yaml:"publisher"`
	// SKU is the image SKU (e.g. Alpha).
	SKU string `json:"sku" yaml:"sku"`
	// Version is the image version (e.g. 1465.7.0).
	Version string `json:"version" yaml:"version"`
}

// newNodeOSImage provides OS information for Container Linux
func newNodeOSImageCoreOS() nodeOSImage {
	return nodeOSImage{
		Offer:     "CoreOS",
		Publisher: "CoreOS",
		SKU:       "Stable",
		Version:   "1967.5.0",
	}
}

// SmallCloudconfigConfig represents the data structure required for executing
// the small cloudconfig template.
type SmallCloudconfigConfig struct {
	BlobURL string
}
