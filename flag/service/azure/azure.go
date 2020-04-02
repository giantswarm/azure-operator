package azure

import (
	"github.com/giantswarm/azure-operator/flag/service/azure/hostcluster"
	"github.com/giantswarm/azure-operator/flag/service/azure/msi"
	"github.com/giantswarm/azure-operator/flag/service/azure/template"
)

type Azure struct {
	ClientID         string
	ClientSecret     string
	EnvironmentName  string
	HostCluster      hostcluster.HostCluster
	MSI              msi.MSI
	Location         string
	SubscriptionID   string
	TenantID         string
	Template         template.Template
	VMSSCheckWorkers string
}
