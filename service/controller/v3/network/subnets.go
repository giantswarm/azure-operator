package network

import (
	"bytes"
	"net"
)

// Subnets holds network subnets used by guest clusters.
type Subnets struct {
	Parent net.IPNet
	Calico net.IPNet
	Master net.IPNet
	VPN    net.IPNet
	Worker net.IPNet
}

// Equal return true when every network in a and b are equal.
func (a Subnets) Equal(b Subnets) bool {
	return a.Calico.IP.Equal(b.Calico.IP) &&
		bytes.Equal(a.Calico.Mask, b.Calico.Mask) &&
		a.Master.IP.Equal(b.Master.IP) &&
		bytes.Equal(a.Master.Mask, b.Master.Mask) &&
		a.Parent.IP.Equal(b.Parent.IP) &&
		bytes.Equal(a.Parent.Mask, b.Parent.Mask) &&
		a.VPN.IP.Equal(b.VPN.IP) &&
		bytes.Equal(a.VPN.Mask, b.VPN.Mask) &&
		a.Worker.IP.Equal(b.Worker.IP) &&
		bytes.Equal(a.Worker.Mask, b.Worker.Mask)
}
