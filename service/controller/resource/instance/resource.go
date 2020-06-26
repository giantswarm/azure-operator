package instance

import (
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"k8s.io/client-go/kubernetes"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v4/service/controller/resource/nodes"
)

const (
	Name = "instance"
)

type Config struct {
	nodes.Config
	CtrlClient                ctrlclient.Client
	GSClientCredentialsConfig auth.ClientCredentialsConfig
}

type Resource struct {
	nodes.Resource
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
