package azure

import "github.com/giantswarm/azuretpr/spec/azure/node"

type Node struct {
	// VMSize is the master vm size (e.g. Standard_A1)
	VMSize string `json:"vmSize" yaml:"vmSize"`
	// AdminUsername is the vm administrator username
	AdminUsername string `json:"adminUsername" yaml:"adminUsername"`
	//  AdminSSHKeyData is the vm administrator ssh public key
	AdminSSHKeyData string `json:"adminSSHKeyData" yaml:"adminSSHKeyData"`
	// OSImage is the vm OS image object
	OSImage node.OSImage `json:"osImage" yaml:"osImage"`
}
