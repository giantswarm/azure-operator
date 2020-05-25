package installation

import (
	"github.com/giantswarm/azure-operator/v4/flag/service/installation/guest"
	"github.com/giantswarm/azure-operator/v4/flag/service/installation/tenant"
)

type Installation struct {
	Name   string
	Guest  guest.Guest
	Tenant tenant.Tenant
}
