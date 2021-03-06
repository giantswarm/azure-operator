package network

type Network struct {
	// CIDR is network segment from which IPAM allocates subnets for guest
	// clusters.
	CIDR string

	// SubnetMaskBits is number of bits in guest cluster subnet mask. This
	// defines size of the guest cluster subnet that is allocated from CIDR.
	SubnetMaskBits string
}
