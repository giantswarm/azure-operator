package azure

import "github.com/giantswarm/azure-operator/flag/service/azure/template"

type Azure struct {
	ClientID       string
	ClientSecret   string
	SubscriptionID string
	TenantID       string
	Template       template.Template
}
