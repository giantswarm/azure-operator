package tenant

import (
	"github.com/giantswarm/azure-operator/v7/flag/service/installation/tenant/kubernetes"
)

type Tenant struct {
	Kubernetes kubernetes.Kubernetes
}
