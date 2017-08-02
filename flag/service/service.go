package service

import (
	"github.com/giantswarm/azure-operator/flag/service/kubernetes"
)

type Service struct {
	Kubernetes kubernetes.Kubernetes
}
