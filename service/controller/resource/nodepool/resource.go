package nodepool

import (
	"context"

	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/giantswarm/errors/tenant"
	"github.com/giantswarm/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/tenantcluster/v2/pkg/tenantcluster"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v4/pkg/credential"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/nodes"
)

const (
	Name = "nodepool"
)

type Config struct {
	nodes.Config
	CredentialProvider        credential.Provider
	CtrlClient                ctrlclient.Client
	GSClientCredentialsConfig auth.ClientCredentialsConfig
	TenantRestConfigProvider  tenantcluster.Interface
}

// Resource takes care of node pool life cycle.
type Resource struct {
	nodes.Resource
	CredentialProvider        credential.Provider
	CtrlClient                ctrlclient.Client
	GSClientCredentialsConfig auth.ClientCredentialsConfig
	k8sClient                 kubernetes.Interface
	tenantRestConfigProvider  tenantcluster.Interface
}

func New(config Config) (*Resource, error) {
	r := &Resource{
		Resource: nodes.Resource{
			Logger:           config.Logger,
			Debugger:         config.Debugger,
			G8sClient:        config.G8sClient,
			Azure:            config.Azure,
			ClientFactory:    config.ClientFactory,
			InstanceWatchdog: config.InstanceWatchdog,
		},
		CredentialProvider:        config.CredentialProvider,
		CtrlClient:                config.CtrlClient,
		GSClientCredentialsConfig: config.GSClientCredentialsConfig,
		k8sClient:                 config.K8sClient,
		tenantRestConfigProvider:  config.TenantRestConfigProvider,
	}
	stateMachine := r.createStateMachine()
	r.SetStateMachine(stateMachine)

	return r, nil
}

func (r *Resource) Name() string {
	return Name
}

func (r *Resource) getTenantClusterK8sClient(ctx context.Context, cluster *capiv1alpha3.Cluster) (k8sclient.Interface, error) {
	var k8sClient k8sclient.Interface
	{
		restConfig, err := r.tenantRestConfigProvider.NewRestConfig(ctx, key.ClusterID(cluster), cluster.Spec.ControlPlaneEndpoint.String())
		if tenantcluster.IsTimeout(err) {
			r.Logger.LogCtx(ctx, "level", "debug", "message", "timeout fetching certificates")
			r.Logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
			return k8sClient, nil
		} else if err != nil {
			return k8sClient, microerror.Mask(err)
		}

		c := k8sclient.ClientsConfig{
			Logger:     r.Logger,
			RestConfig: rest.CopyConfig(restConfig),
		}

		k8sClient, err := k8sclient.NewClients(c)
		if tenant.IsAPINotAvailable(err) {
			r.Logger.LogCtx(ctx, "level", "debug", "message", "tenant API not available yet")
			r.Logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
			return k8sClient, nil
		} else if err != nil {
			return k8sClient, microerror.Mask(err)
		}
	}

	return k8sClient, nil
}
