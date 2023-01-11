package tenantcluster

import (
	"context"
	"time"

	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	gocache "github.com/patrickmn/go-cache"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v7/service/controller/key"
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

func (ctcf *cachedTenantClientFactory) GetAllClients(ctx context.Context, cr *capi.Cluster) (k8sclient.Interface, error) {
	ctcf.logger.Debugf(ctx, "trying to fetch tenant cluster %#q k8s client from cache before creating it", key.ClusterID(cr))
	tenantClusterClient, inCache := ctcf.cache.Get(key.ClusterID(cr))
	if inCache {
		ctcf.logger.Debugf(ctx, "tenant cluster k8s client for cluster %#q found in cache", key.ClusterID(cr))
		return tenantClusterClient.(k8sclient.Interface), nil
	}

	tenantClusterClient, err := ctcf.factory.GetAllClients(ctx, cr)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	ctcf.cache.SetDefault(key.ClusterID(cr), tenantClusterClient)
	ctcf.logger.Debugf(ctx, "saved tenant cluster k8s client for cluster %#q in cache", key.ClusterID(cr))

	return tenantClusterClient.(k8sclient.Interface), nil
}

func (ctcf *cachedTenantClientFactory) GetClient(ctx context.Context, cr *capi.Cluster) (client.Client, error) {
	all, err := ctcf.GetAllClients(ctx, cr)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return all.CtrlClient(), nil
}
