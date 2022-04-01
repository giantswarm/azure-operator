package masters

import (
	"context"

	providerv1alpha1 "github.com/giantswarm/apiextensions/v6/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/tenantcluster/v5/pkg/tenantcluster"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v5/pkg/handler/nodes"
	"github.com/giantswarm/azure-operator/v5/service/controller/internal/vmsku"
	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

const (
	Name = "masters"
)

type Config struct {
	nodes.Config
	CtrlClient               client.Client
	TenantRestConfigProvider *tenantcluster.TenantCluster
	VMSKU                    *vmsku.VMSKUs
}

type Resource struct {
	nodes.Resource
	ctrlClient               client.Client
	tenantRestConfigProvider *tenantcluster.TenantCluster
	vmSku                    *vmsku.VMSKUs
}

func New(config Config) (*Resource, error) {
	if config.CtrlClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CtrlClient must not be empty", config)
	}
	if config.TenantRestConfigProvider == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.TenantRestConfigProvider must not be empty", config)
	}
	if config.VMSKU == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.VMSKU must not be empty", config)
	}

	config.Name = Name
	nodes, err := nodes.New(config.Config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	r := &Resource{
		Resource:                 *nodes,
		ctrlClient:               config.CtrlClient,
		tenantRestConfigProvider: config.TenantRestConfigProvider,
		vmSku:                    config.VMSKU,
	}
	stateMachine := r.createStateMachine()
	r.SetStateMachine(stateMachine)

	return r, nil
}

func (r *Resource) Name() string {
	return r.Resource.Name()
}

func (r *Resource) getTenantClusterClient(ctx context.Context, azureConfig *providerv1alpha1.AzureConfig) (client.Client, error) {
	var k8sClient k8sclient.Interface
	{
		restConfig, err := r.tenantRestConfigProvider.NewRestConfig(ctx, key.ClusterID(azureConfig), key.ClusterAPIEndpoint(*azureConfig))
		if err != nil {
			return nil, microerror.Mask(err)
		}

		k8sClient, err = k8sclient.NewClients(k8sclient.ClientsConfig{
			Logger:     r.Logger,
			RestConfig: rest.CopyConfig(restConfig),
		})
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return k8sClient.CtrlClient(), nil
}
