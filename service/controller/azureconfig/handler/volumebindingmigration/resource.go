package volumebindingmigration

import (
	"context"

	providerv1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/tenantcluster/v3/pkg/tenantcluster"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

const (
	// Name is the identifier of the resource.
	Name = "volumebindingmigration"
)

type Config struct {
	Logger                   micrologger.Logger
	TenantRestConfigProvider *tenantcluster.TenantCluster
}

// Resource ensures that existing StorageClasses use `WaitForFirstConsumer`
// bind mode.
type Resource struct {
	logger                   micrologger.Logger
	tenantRestConfigProvider *tenantcluster.TenantCluster
}

func New(config Config) (*Resource, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.TenantRestConfigProvider == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.TenantRestConfigProvider must not be empty", config)
	}

	r := &Resource{
		logger:                   config.Logger,
		tenantRestConfigProvider: config.TenantRestConfigProvider,
	}

	return r, nil
}

// EnsureCreated ensures that existing StorageClasses use
// `WaitForFirstConsumer` bind mode.
func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	var tenantClusterK8sClient client.Client
	{
		tenantClusterK8sClient, err = r.getTenantClusterClient(ctx, &cr)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	desiredVolumeBindMode := storagev1.VolumeBindingWaitForFirstConsumer

	r.logger.Debugf(ctx, "ensuring storageclasses use desired volumeBindMode %q", desiredVolumeBindMode)

	storageClassList := &storagev1.StorageClassList{}
	err = tenantClusterK8sClient.List(ctx, storageClassList)
	if err != nil {
		return microerror.Mask(err)
	}

	// Iterate over StorageClasses and remove items that already have expected
	// volume binding mode. Rest will be updated in one call.
	for i := 0; i < len(storageClassList.Items); i++ {
		sc := storageClassList.Items[i]

		if sc.VolumeBindingMode != nil && *sc.VolumeBindingMode == storagev1.VolumeBindingWaitForFirstConsumer {
			storageClassList.Items = append(storageClassList.Items[:i], storageClassList.Items[i+1:]...)
			i--
		} else {
			storageClassList.Items[i].VolumeBindingMode = &desiredVolumeBindMode
		}
	}

	if len(storageClassList.Items) > 0 {
		err = tenantClusterK8sClient.Update(ctx, storageClassList)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	r.logger.Debugf(ctx, "ensured storageclasses use desired volumeBindMode %q", desiredVolumeBindMode)

	return nil
}

// EnsureDeleted is no-op.
func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	return nil
}

// Name returns the resource name.
func (r *Resource) Name() string {
	return Name
}

func (r *Resource) getTenantClusterClient(ctx context.Context, azureConfig *providerv1alpha1.AzureConfig) (client.Client, error) {
	var k8sClient k8sclient.Interface
	{
		restConfig, err := r.tenantRestConfigProvider.NewRestConfig(ctx, key.ClusterID(azureConfig), key.ClusterAPIEndpoint(*azureConfig))
		if err != nil {
			return nil, microerror.Mask(err)
		}

		k8sClient, err = k8sclient.NewClients(k8sclient.ClientsConfig{
			Logger:     r.logger,
			RestConfig: rest.CopyConfig(restConfig),
		})
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return k8sClient.CtrlClient(), nil
}
