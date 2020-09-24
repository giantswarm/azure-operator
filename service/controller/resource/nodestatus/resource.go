package nodestatus

import (
	"context"

	"github.com/giantswarm/k8sclient/v2/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/tenantcluster/v3/pkg/tenantcluster"
	"k8s.io/client-go/rest"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

const (
	// Name is the identifier of the resource.
	Name = "nodestatus"
)

type Config struct {
	CtrlClient               ctrlclient.Client
	Logger                   micrologger.Logger
	TenantRestConfigProvider tenantcluster.Interface
}

// Resource updates the MachinePool status field with the Nodes status.
type Resource struct {
	ctrlClient               ctrlclient.Client
	logger                   micrologger.Logger
	tenantRestConfigProvider tenantcluster.Interface
}

func New(config Config) (*Resource, error) {
	if config.CtrlClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CtrlClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.TenantRestConfigProvider == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.TenantRestConfigProvider must not be empty", config)
	}

	r := &Resource{
		ctrlClient:               config.CtrlClient,
		logger:                   config.Logger,
		tenantRestConfigProvider: config.TenantRestConfigProvider,
	}

	return r, nil
}

func (r *Resource) Name() string {
	return Name
}

func (r *Resource) getTenantClusterK8sClient(ctx context.Context, cluster *capiv1alpha3.Cluster) (k8sclient.Interface, error) {
	var k8sClient k8sclient.Interface
	{
		restConfig, err := r.tenantRestConfigProvider.NewRestConfig(ctx, key.ClusterID(cluster), cluster.Spec.ControlPlaneEndpoint.String())
		if err != nil {
			return k8sClient, microerror.Mask(err)
		}

		k8sClient, err = k8sclient.NewClients(k8sclient.ClientsConfig{
			Logger:     r.logger,
			RestConfig: rest.CopyConfig(restConfig),
		})
		if err != nil {
			return k8sClient, microerror.Mask(err)
		}
	}

	return k8sClient, nil
}
