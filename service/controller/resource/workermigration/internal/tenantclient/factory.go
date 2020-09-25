package tenantclient

import (
	"context"

	"github.com/giantswarm/certs/v3/pkg/certs"
	"github.com/giantswarm/k8sclient/v4/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/tenantcluster/v3/pkg/tenantcluster"
	"k8s.io/client-go/rest"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

type tenantClientFactory struct {
	logger                   micrologger.Logger
	tenantRestConfigProvider tenantcluster.Interface
}

func NewFactory(certsSearcher certs.Interface, logger micrologger.Logger) (Factory, error) {
	c := tenantcluster.Config{
		CertsSearcher: certsSearcher,
		Logger:        logger,

		CertID: certs.APICert,
	}

	tenantRestConfigProvider, err := tenantcluster.New(c)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	f := &tenantClientFactory{
		logger:                   logger,
		tenantRestConfigProvider: tenantRestConfigProvider,
	}

	return f, nil
}

func (tcf *tenantClientFactory) GetClient(ctx context.Context, cr *capiv1alpha3.Cluster) (client.Client, error) {
	var k8sClient k8sclient.Interface
	{
		restConfig, err := tcf.tenantRestConfigProvider.NewRestConfig(ctx, key.ClusterID(cr), cr.Spec.ControlPlaneEndpoint.String())
		if err != nil {
			return nil, microerror.Mask(err)
		}

		k8sClient, err = k8sclient.NewClients(k8sclient.ClientsConfig{
			Logger:     tcf.logger,
			RestConfig: rest.CopyConfig(restConfig),
		})
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return k8sClient.CtrlClient(), nil
}
