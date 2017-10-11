package node

type OSImage struct {
	// Publisher is the image publisher (e.g GiantSwarm)
	Publisher string `json:"publisher" yaml:"publisher"`
	// Offer is the image offered by the publisher (e.g. CoreOS)
	Offer string `json:"offer" yaml:"offer"`
	// SKU is the image SKU (e.g. Alpha)
	SKU string `json:"sku" yaml:"sku"`
	// Version is the image version (e.g. 1465.7.0)
	Version string `json:"version" yaml:"version"`
}
