package kubectl

import (
	"github.com/giantswarm/azure-operator/v4/flag/service/cluster/kubernetes/kubectl/docker"
)

type Kubectl struct {
	Docker docker.Docker
}
