package tcnp

import (
	"context"

	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	v1alpha32 "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v4/service/controller/debugger"
)

const (
	// Name is the identifier of the resource.
	Name = "tcnp"
)

type Config struct {
	CtrlClient                client.Client
	Debugger                  *debugger.Debugger
	GSClientCredentialsConfig auth.ClientCredentialsConfig
	Logger                    micrologger.Logger
	VMSSMSIEnabled            bool
}

type Resource struct {
	ctrlClient                client.Client
	debugger                  *debugger.Debugger
	gsClientCredentialsConfig auth.ClientCredentialsConfig
	logger                    micrologger.Logger
	vmssMSIEnabled            bool
}

func New(config Config) (*Resource, error) {
	if config.CtrlClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CtrlClient must not be empty", config)
	}
	if config.Debugger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Debugger must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	r := &Resource{
		ctrlClient:                config.CtrlClient,
		debugger:                  config.Debugger,
		gsClientCredentialsConfig: config.GSClientCredentialsConfig,
		logger:                    config.Logger,
		vmssMSIEnabled:            config.VMSSMSIEnabled,
	}

	return r, nil
}

// Name returns the resource name.
func (r *Resource) Name() string {
	return Name
}

func (r *Resource) getResourceStatus(ctx context.Context, cr v1alpha32.AzureMachinePool, t string) (string, error) {
	azureMachinePool := &v1alpha32.AzureMachinePool{}
	err := r.ctrlClient.Get(ctx, client.ObjectKey{Namespace: cr.Namespace, Name: cr.Name}, azureMachinePool)
	if err != nil {
		return "", microerror.Mask(err)
	}

	return "", nil
}

func (r *Resource) setResourceStatus(ctx context.Context, cr v1alpha32.AzureMachinePool, t string, s string) error {
	azureMachinePool := &v1alpha32.AzureMachinePool{}
	err := r.ctrlClient.Get(ctx, client.ObjectKey{Namespace: cr.GetNamespace(), Name: cr.GetName()}, azureMachinePool)
	if err != nil {
		return microerror.Mask(err)
	}

	{
		err := r.ctrlClient.Status().Update(ctx, &cr)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}
