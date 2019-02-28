package installation

import (
	"github.com/giantswarm/azure-operator/flag/service/installation/tenant"
)

type Installation struct {
	Name   string
	Tenant tenant.Tenant
}
