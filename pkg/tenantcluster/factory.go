package tenantcluster

import (
	"context"
	"fmt"

	"github.com/giantswarm/certs/v3/pkg/certs"
	"github.com/giantswarm/errors/tenant"
	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/tenantcluster/v5/pkg/tenantcluster"
	"k8s.io/client-go/rest"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v5/pkg/project"
	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

type tenantClientFactory struct {
	logger                   micrologger.Logger
	tenantRestConfigProvider tenantcluster.Interface
}

func NewFactory(certsSearcher certs.Interface, logger micrologger.Logger) (Factory, error) {
	if certsSearcher == nil {
		return nil, microerror.Maskf(invalidConfigError, "certsSearcher must not be empty")
	}
	if logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "logger must not be empty")
	}

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

func (tcf *tenantClientFactory) GetAllClients(ctx context.Context, cr *capi.Cluster) (k8sclient.Interface, error) {
	tcf.logger.Debugf(ctx, "creating tenant cluster k8s client for cluster %#q", key.ClusterID(cr))
	var k8sClient k8sclient.Interface
	{
		restConfig, err := tcf.tenantRestConfigProvider.NewRestConfig(ctx, key.ClusterID(cr), cr.Spec.ControlPlaneEndpoint.String())
		if tenant.IsAPINotAvailable(err) || tenantcluster.IsTimeout(err) {
			return nil, microerror.Mask(apiNotAvailableError)
		} else if err != nil {
			return nil, microerror.Mask(err)
		}

		restConfig.UserAgent = fmt.Sprintf("%s/%s", project.Name(), project.Version())

		k8sClient, err = k8sclient.NewClients(k8sclient.ClientsConfig{
			Logger:     tcf.logger,
			RestConfig: rest.CopyConfig(restConfig),
		})
		if tenant.IsAPINotAvailable(err) {
			return nil, microerror.Mask(apiNotAvailableError)
		} else if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return k8sClient, nil
}

func (tcf *tenantClientFactory) GetClient(ctx context.Context, cr *capi.Cluster) (client.Client, error) {
	all, err := tcf.GetAllClients(ctx, cr)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	return all.CtrlClient(), nil
}
