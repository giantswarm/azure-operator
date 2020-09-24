package tenantcluster

import (
	"context"

	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//go:generate mockgen -destination ../mock/mock_tenantcluster/factory.go -source spec.go Factory

type Factory interface {
	GetClient(ctx context.Context, cr *capiv1alpha3.Cluster) (client.Client, error)
}
