package tenant

import (
	"github.com/giantswarm/azure-operator/flag/service/tenant/ignition"
	"github.com/giantswarm/azure-operator/flag/service/tenant/ssh"
)

type Tenant struct {
	Ignition ignition.Ignition
	SSH      ssh.SSH
}
