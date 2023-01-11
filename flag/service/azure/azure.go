package azure

import (
	"github.com/giantswarm/azure-operator/v7/flag/service/azure/hostcluster"
	"github.com/giantswarm/azure-operator/v7/flag/service/azure/msi"
	"github.com/giantswarm/azure-operator/v7/flag/service/azure/template"
)

type Azure struct {
	ClientID        string
	ClientSecret    string
	EnvironmentName string
	HostCluster     hostcluster.HostCluster
	MSI             msi.MSI
	Location        string
	PartnerID       string
	SubscriptionID  string
	TenantID        string
	Template        template.Template
}
