package tenant

import (
	"github.com/giantswarm/azure-operator/flag/service/installation/tenant/kubernetes"
)

type Tenant struct {
	Kubernetes kubernetes.Kubernetes
}
