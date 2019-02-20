package guest

import (
	"github.com/giantswarm/azure-operator/flag/service/guest/ignition"
	"github.com/giantswarm/azure-operator/flag/service/guest/ssh"
)

type Guest struct {
	Ignition ignition.Ignition
	SSH      ssh.SSH
}
