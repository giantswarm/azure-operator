package tenantclient

import (
	"context"

	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Factory interface {
	GetClient(ctx context.Context, cr *capiv1alpha3.Cluster) (client.Client, error)
}
