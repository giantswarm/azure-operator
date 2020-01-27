package controllercontext

import (
	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ContextClient struct {
	TenantCluster ContextClientTenantCluster
}

type ContextClientTenantCluster struct {
	G8s        versioned.Interface
	K8s        kubernetes.Interface
	CtrlClient client.Client
}
