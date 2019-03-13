package service

import (
	"github.com/giantswarm/operatorkit/flag/service/kubernetes"

	"github.com/giantswarm/azure-operator/flag/service/azure"
	"github.com/giantswarm/azure-operator/flag/service/installation"
	"github.com/giantswarm/azure-operator/flag/service/tenant"
)

type Service struct {
	Azure        azure.Azure
	Installation installation.Installation
	Kubernetes   kubernetes.Kubernetes
	Tenant       tenant.Tenant
}
