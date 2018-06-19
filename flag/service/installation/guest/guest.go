package guest

import (
	"github.com/giantswarm/azure-operator/flag/service/installation/guest/kubernetes"
)

type Guest struct {
	Kubernetes kubernetes.Kubernetes
}
