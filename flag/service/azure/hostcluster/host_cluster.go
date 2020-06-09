package hostcluster

import (
	"github.com/giantswarm/azure-operator/v4/flag/service/azure/hostcluster/tenant"
)

type HostCluster struct {
	CIDR                  string
	ResourceGroup         string
	Tenant                tenant.Tenant
	VirtualNetwork        string
	VirtualNetworkGateway string
}
