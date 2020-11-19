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

func NewCachedFactory(tenantClientFactory Factory, logger micrologger.Logger) (Factory, error) {
	if tenantClientFactory == nil {
		return nil, microerror.Maskf(invalidConfigError, "tenantClientFactory must not be empty")
	}
	if logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "logger must not be empty")
	}

	return &cachedTenantClientFactory{
		cache:   gocache.New(expiration, expiration/2),
		logger:  logger,
		factory: tenantClientFactory,
	}, nil
}

func (ctcf *cachedTenantClientFactory) GetClient(ctx context.Context, cr *capiv1alpha3.Cluster) (client.Client, error) {
	ctcf.logger.LogCtx(ctx, "level", "debug", "message", "Fetching tenant cluster k8s client for cluster %#q from cache", key.ClusterID(cr))
	tenantClusterClient, inCache := ctcf.cache.Get(key.ClusterID(cr))
	if inCache {
		ctcf.logger.LogCtx(ctx, "level", "debug", "message", "Tenant cluster k8s client for cluster %#q found in cache", key.ClusterID(cr))
		return tenantClusterClient.(client.Client), nil
	}

	tenantClusterClient, err := ctcf.factory.GetClient(ctx, cr)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	ctcf.cache.SetDefault(key.ClusterID(cr), tenantClusterClient)
	ctcf.logger.LogCtx(ctx, "level", "debug", "message", "Saved tenant cluster k8s client for cluster %#q in cache", key.ClusterID(cr))

	return tenantClusterClient.(client.Client), nil
}
