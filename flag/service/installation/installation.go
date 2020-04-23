package installation

import (
	"github.com/giantswarm/azure-operator/v3/flag/service/installation/tenant"
)

type Installation struct {
	Name   string
	Tenant tenant.Tenant
}
