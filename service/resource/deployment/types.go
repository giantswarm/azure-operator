package deployment

import providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"

func newNodes(azureNodes []providerv1alpha1.AzureConfigSpecAzureNode) []node {
	var ns []node
	for _, n := range azureNodes {
		ns = append(ns, newNode(n))
	}
	return ns
}

type node struct {
	providerv1alpha1.AzureConfigSpecAzureNode

	// OSImage is the vm OS image object.
	OSImage nodeOSImage `json:"osImage" yaml:"osImage"`
}

func newNode(azureNode providerv1alpha1.AzureConfigSpecAzureNode) node {
	return node{
		AzureConfigSpecAzureNode: azureNode,
		OSImage:                  newNodeOSImage(),
	}
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

// newNodeOSImage provides OS information CoreOS 1465.7.0.
func newNodeOSImage() nodeOSImage {
	return nodeOSImage{
		Offer:     "CoreOS",
		Publisher: "CoreOS",
		SKU:       "Stable",
		Version:   "1465.7.0",
	}
}
