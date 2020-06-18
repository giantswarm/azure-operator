package networksetup

import (
	"github.com/giantswarm/azure-operator/v4/flag/service/cluster/kubernetes/networksetup/docker"
)

type NetworkSetup struct {
	Docker docker.Docker
}
