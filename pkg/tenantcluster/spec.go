package tenantcluster

import (
	"context"

	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"

	capi "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//go:generate mockgen -destination ../mock/mock_tenantcluster/factory.go -source spec.go Factory

type Factory interface {
	GetAllClients(ctx context.Context, cr *capi.Cluster) (k8sclient.Interface, error)
	GetClient(ctx context.Context, cr *capi.Cluster) (client.Client, error)
}
