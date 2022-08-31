package docker

import (
	"github.com/giantswarm/azure-operator/v6/flag/service/cluster/docker/daemon"
)

type Docker struct {
	Daemon daemon.Daemon
}
