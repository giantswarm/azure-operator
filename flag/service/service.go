package service

import (
	"github.com/giantswarm/operatorkit/flag/service/kubernetes"

	"github.com/giantswarm/azure-operator/v3/flag/service/azure"
	"github.com/giantswarm/azure-operator/v3/flag/service/installation"
	"github.com/giantswarm/azure-operator/v3/flag/service/tenant"
)

type Service struct {
	Azure          azure.Azure
	Installation   installation.Installation
	Kubernetes     kubernetes.Kubernetes
	RegistryDomain string
	Tenant         tenant.Tenant
}
