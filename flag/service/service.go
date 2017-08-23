package service

import (
	"github.com/giantswarm/azure-operator/flag/service/azure"
	"github.com/giantswarm/azure-operator/flag/service/kubernetes"
)

type Service struct {
	Azure      azure.Azure
	Kubernetes kubernetes.Kubernetes
}
