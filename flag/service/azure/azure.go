package azure

import (
	"github.com/giantswarm/azure-operator/flag/service/azure/hostcluster"
	"github.com/giantswarm/azure-operator/flag/service/azure/template"
)

type Azure struct {
	ClientID       string
	ClientSecret   string
	HostCluster    hostcluster.HostCluster
	Location       string
	SubscriptionID string
	TenantID       string
	Template       template.Template
}
