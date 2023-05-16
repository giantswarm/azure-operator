package tenant

import (
	"github.com/giantswarm/azure-operator/v8/flag/service/installation/tenant/kubernetes"
)

type Tenant struct {
	Kubernetes kubernetes.Kubernetes
}
