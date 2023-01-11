package tenant

import (
	"github.com/giantswarm/azure-operator/v7/flag/service/tenant/ignition"
	"github.com/giantswarm/azure-operator/v7/flag/service/tenant/ssh"
)

type Tenant struct {
	Ignition ignition.Ignition
	SSH      ssh.SSH
}
