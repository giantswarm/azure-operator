package tenant

import (
	"github.com/giantswarm/azure-operator/v5/flag/service/installation/tenant/kubernetes"
)

type Tenant struct {
	Kubernetes kubernetes.Kubernetes
}
