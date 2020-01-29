package setting

import "fmt"

type Azure struct {
	EnvironmentName string
	HostCluster     AzureHostCluster
	MSI             AzureMSI
	Location        string
}

func (a Azure) Validate() error {
	if a.EnvironmentName == "" {
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
	CIDR                  string
	ResourceGroup         string
	VirtualNetwork        string
	VirtualNetworkGateway string
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
	if h.VirtualNetworkGateway == "" {
		return fmt.Errorf("VirtualNetworkGateway must not be empty")
	}

	return nil
}

type AzureMSI struct {
	Enabled bool
}

type Ignition struct {
	Path       string
	Debug      bool
	LogsPrefix string
	LogsToken  string
}

type OIDC struct {
	ClientID      string
	IssuerURL     string
	UsernameClaim string
	GroupsClaim   string
}
