package azure

import (
	"github.com/giantswarm/azure-operator/flag/service/azure/hostcluster"
	"github.com/giantswarm/azure-operator/flag/service/azure/msi"
	"github.com/giantswarm/azure-operator/flag/service/azure/network"
	"github.com/giantswarm/azure-operator/flag/service/azure/template"
)

type Azure struct {
	Cloud          string
	ClientID       string
	ClientSecret   string
	HostCluster    hostcluster.HostCluster
	MSI            msi.MSI
	Network        network.Network
	Location       string
	SubscriptionID string
	TenantID       string
	Template       template.Template
}
