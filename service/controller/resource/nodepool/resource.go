package nodepool

import (
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"k8s.io/client-go/kubernetes"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v4/pkg/credential"
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
}

// Resource takes care of node pool life cycle.
type Resource struct {
	nodes.Resource
	CredentialProvider        credential.Provider
	CtrlClient                ctrlclient.Client
	GSClientCredentialsConfig auth.ClientCredentialsConfig
	k8sClient                 kubernetes.Interface
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
	}
	stateMachine := r.createStateMachine()
	r.SetStateMachine(stateMachine)

	return r, nil
}

func (r *Resource) Name() string {
	return Name
}
