package controllercontext

import (
	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"k8s.io/client-go/kubernetes"
)

type ContextClient struct {
	TenantCluster ContextClientTenantCluster
}

type ContextClientTenantCluster struct {
	G8s versioned.Interface
	K8s kubernetes.Interface
}
