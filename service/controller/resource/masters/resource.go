package masters

import (
	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/azure-operator/service/controller/debugger"
	"github.com/giantswarm/azure-operator/service/controller/internal/state"
	"github.com/giantswarm/azure-operator/service/controller/internal/vmsscheck"
	"github.com/giantswarm/azure-operator/service/controller/setting"
)

const (
	Name = "masters"
)

type Config struct {
	Debugger  *debugger.Debugger
	G8sClient versioned.Interface
	K8sClient kubernetes.Interface
	Logger    micrologger.Logger

	Azure            setting.Azure
	InstanceWatchdog vmsscheck.InstanceWatchdog
	TemplateVersion  string
}

type Resource struct {
	debugger     *debugger.Debugger
	g8sClient    versioned.Interface
	k8sClient    kubernetes.Interface
	logger       micrologger.Logger
	stateMachine state.Machine

	azure            setting.Azure
	instanceWatchdog vmsscheck.InstanceWatchdog
	templateVersion  string
}

func New(config Config) (*Resource, error) {
	if config.Debugger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Debugger must not be empty", config)
	}
	if config.G8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.G8sClient must not be empty", config)
	}
	if config.InstanceWatchdog == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.InstanceWatchdog must not be empty", config)
	}
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if err := config.Azure.Validate(); err != nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Azure.%s", config, err)
	}
	if config.TemplateVersion == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.TemplateVersion must not be empty", config)
	}

	r := &Resource{
		debugger:  config.Debugger,
		g8sClient: config.G8sClient,
		k8sClient: config.K8sClient,
		logger:    config.Logger,

		azure:            config.Azure,
		instanceWatchdog: config.InstanceWatchdog,
		templateVersion:  config.TemplateVersion,
	}

	r.configureStateMachine()

	return r, nil
}

func (r *Resource) Name() string {
	return Name
}
