package tenantcluster

import (
	"context"
	"time"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	gocache "github.com/patrickmn/go-cache"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

const (
	expiration = 5 * time.Minute
)

type cachedTenantClientFactory struct {
	cache   *gocache.Cache
	logger  micrologger.Logger
	factory Factory
}

func NewCachedFactory(tenantClientFactory Factory, logger micrologger.Logger) Factory {
	return &cachedTenantClientFactory{
		cache:   gocache.New(expiration, expiration/2),
		logger:  logger,
		factory: tenantClientFactory,
	}
}

func (ctcf *cachedTenantClientFactory) GetClient(ctx context.Context, cr *capiv1alpha3.Cluster) (client.Client, error) {
	tenantClusterClient, inCache := ctcf.cache.Get(key.ClusterID(cr))
	if inCache {
		return tenantClusterClient.(client.Client), nil
	}

	tenantClusterClient, err := ctcf.factory.GetClient(ctx, cr)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	ctcf.cache.SetDefault(key.ClusterID(cr), tenantClusterClient)

	return tenantClusterClient.(client.Client), nil
}
