package setting

import "fmt"

type Azure struct {
	Cloud       string
	HostCluster AzureHostCluster
	MSI         AzureMSI
	Network     AzureNetwork
	Location    string
}

func (a Azure) Validate() error {
	if a.Cloud == "" {
		return fmt.Errorf("Location must not be empty")
	}
	if err := a.HostCluster.Validate(); err != nil {
		return fmt.Errorf("HostCluster.%s", err)
	}
	if a.Location == "" {
		return fmt.Errorf("Location must not be empty")
	}

	return nil
}

type AzureHostCluster struct {
	CIDR           string
	ResourceGroup  string
	VirtualNetwork string
}

func (h AzureHostCluster) Validate() error {
	if h.CIDR == "" {
		return fmt.Errorf("CIDR must not be empty")
	}
	if h.ResourceGroup == "" {
		return fmt.Errorf("ResourceGroup must not be empty")
	}
	if h.VirtualNetwork == "" {
		return fmt.Errorf("VirtualNetwork must not be empty")
	}

	return nil
}

type AzureMSI struct {
	Enabled bool
}

type OIDC struct {
	ClientID      string
	IssuerURL     string
	UsernameClaim string
	GroupsClaim   string
}

type AzureNetwork struct {
	CalicoSubnetMask int
	MasterSubnetMask int
	VPNSubnetMask    int
	WorkerSubnetMask int
}
