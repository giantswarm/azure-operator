package volumebindingmigration

import (
	"bytes"
	"context"

	providerv1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/tenantcluster/v3/pkg/tenantcluster"
	storagev1 "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/azure-operator/v5/service/controller/key"
	"github.com/giantswarm/azure-operator/v5/service/controller/templates/ignition"
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

	var tenantClusterK8sClient ctrl.Client
	{
		tenantClusterK8sClient, err = r.getTenantClusterClient(ctx, &cr)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	r.logger.Debugf(ctx, "ensuring storageclasses use desired volumeBindingMode")

	defaultSCs, err := defaultStorageClasses()
	if err != nil {
		return microerror.Mask(err)
	}

	for _, desiredObj := range defaultSCs {
		r.logger.Debugf(ctx, "finding present storage class object %q", desiredObj.Name)

		var presentObj storagev1.StorageClass
		err := tenantClusterK8sClient.Get(ctx, ctrl.ObjectKey{Name: desiredObj.Name, Namespace: desiredObj.Namespace}, &presentObj)
		if apierrors.IsNotFound(err) {
			// All good. We'll create it.
			r.logger.Debugf(ctx, "did not find present storage class object %q", desiredObj)
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			r.logger.Debugf(ctx, "finding if present storage class object %q has desired volumeBindingMode %q", desiredObj.Name, desiredObj.VolumeBindingMode)
		}

		// If present object matches the desired one, continue to next one.
		if (desiredObj.VolumeBindingMode == presentObj.VolumeBindingMode) || (desiredObj.VolumeBindingMode != nil && presentObj.VolumeBindingMode != nil && *presentObj.VolumeBindingMode == *desiredObj.VolumeBindingMode) {
			r.logger.Debugf(ctx, "present storage class object %q has desired volumeBindingMode: %q", presentObj.Name, desiredObj.VolumeBindingMode)
			continue
		}

		// Volume bind mode is immutable field so we must delete the present
		// object if it exists.
		if !presentObj.CreationTimestamp.IsZero() && presentObj.ResourceVersion != "" {
			r.logger.Debugf(ctx, "present storage class object %q does not have desired volumeBindingMode but %q instead", presentObj.Name, presentObj.VolumeBindingMode)
			r.logger.Debugf(ctx, "deleting present storage class object %q", presentObj.Name)
			err = tenantClusterK8sClient.Delete(ctx, &presentObj)
			if err != nil {
				return microerror.Mask(err)
			}
			r.logger.Debugf(ctx, "deleted present storage class object %q", presentObj.Name)
		}

		r.logger.Debugf(ctx, "creating desired storage class object %q", desiredObj.Name)

		// Finally create the desired object.
		err = tenantClusterK8sClient.Create(ctx, &desiredObj)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.Debugf(ctx, "created desired storage class object %q", desiredObj.Name)
	}

	r.logger.Debugf(ctx, "ensured storageclasses use desired volumeBindingMode")

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

func (r *Resource) getTenantClusterClient(ctx context.Context, azureConfig *providerv1alpha1.AzureConfig) (ctrl.Client, error) {
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

func defaultStorageClasses() ([]storagev1.StorageClass, error) {
	var storageClasses []storagev1.StorageClass

	objs := bytes.Split([]byte(ignition.DefaultStorageClass), []byte("---"))

	for _, bs := range objs {
		sc := storagev1.StorageClass{}
		err := yaml.Unmarshal(bs, &sc)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		if sc.Kind != "StorageClass" {
			continue
		}

		storageClasses = append(storageClasses, sc)
	}

	return storageClasses, nil
}
