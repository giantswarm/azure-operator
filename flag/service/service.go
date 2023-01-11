package service

import (
	"github.com/giantswarm/operatorkit/v7/pkg/flag/service/kubernetes"

	"github.com/giantswarm/azure-operator/v7/flag/service/azure"
	"github.com/giantswarm/azure-operator/v7/flag/service/cluster"
	"github.com/giantswarm/azure-operator/v7/flag/service/debug"
	"github.com/giantswarm/azure-operator/v7/flag/service/installation"
	"github.com/giantswarm/azure-operator/v7/flag/service/registry"
	"github.com/giantswarm/azure-operator/v7/flag/service/sentry"
	"github.com/giantswarm/azure-operator/v7/flag/service/tenant"
)

type Service struct {
	Azure        azure.Azure
	Cluster      cluster.Cluster
	Installation installation.Installation
	Kubernetes   kubernetes.Kubernetes
	Registry     registry.Registry
	Tenant       tenant.Tenant
	Sentry       sentry.Sentry
	Debug        debug.Debug
}
