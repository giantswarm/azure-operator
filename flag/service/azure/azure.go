package azure

import (
	"github.com/giantswarm/azure-operator/flag/service/azure/hostcluster"
	"github.com/giantswarm/azure-operator/flag/service/azure/msi"
	"github.com/giantswarm/azure-operator/flag/service/azure/template"
)

type Azure struct {
	// TODO rename to EnvironmentName. See https://github.com/giantswarm/giantswarm/issues/4124.
	Cloud          string
	ClientID       string
	ClientSecret   string
	HostCluster    hostcluster.HostCluster
	MSI            msi.MSI
	Location       string
	SubscriptionID string
	TenantID       string
	Template       template.Template
}
