package ingress

import (
	"github.com/giantswarm/azure-operator/v4/flag/service/cluster/kubernetes/ingress/docker"
)

type IngressController struct {
	BaseDomain   string
	Docker       docker.Docker
	InsecurePort string
	SecurePort   string
}
