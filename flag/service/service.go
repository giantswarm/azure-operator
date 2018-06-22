package service

import (
	"github.com/giantswarm/azure-operator/flag/service/azure"
	"github.com/giantswarm/azure-operator/flag/service/guest"
	"github.com/giantswarm/azure-operator/flag/service/installation"
	"github.com/giantswarm/azure-operator/flag/service/kubernetes"
)

type Service struct {
	Azure        azure.Azure
	Guest        guest.Guest
	Installation installation.Installation
	Kubernetes   kubernetes.Kubernetes
}
