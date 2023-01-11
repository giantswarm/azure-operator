package azureconfig

import (
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v5client "github.com/giantswarm/azure-operator/v7/client"
)

const (
	Name = "azureconfig"
)

type CalicoConfig struct {
	CIDRSize int
	MTU      int
	Subnet   string
}

type Config struct {
	CtrlClient client.Client
	Logger     micrologger.Logger

	APIServerSecurePort            int
	Calico                         CalicoConfig
	ClientFactory                  v5client.OrganizationFactory
	ClusterIPRange                 string
	EtcdPrefix                     string
	ManagementClusterResourceGroup string
	VnetMaskSize                   int
}

type Resource struct {
	ctrlClient client.Client
	logger     micrologger.Logger

	apiServerSecurePort            int
	calico                         CalicoConfig
	clientFactory                  v5client.OrganizationFactory
	clusterIPRange                 string
	etcdPrefix                     string
	managementClusterResourceGroup string
	vnetMaskSize                   int
}

func New(config Config) (*Resource, error) {
	if config.CtrlClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CtrlClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.VnetMaskSize == 0 {
		return nil, microerror.Maskf(invalidConfigError, "%T.VnetMaskSize must be set", config)
	}
	// No validation for configuration at this point. I'm not fully sure if any
	// of that is actually needed.

	newResource := &Resource{
		ctrlClient: config.CtrlClient,
		logger:     config.Logger,

		apiServerSecurePort:            config.APIServerSecurePort,
		calico:                         config.Calico,
		clientFactory:                  config.ClientFactory,
		clusterIPRange:                 config.ClusterIPRange,
		etcdPrefix:                     config.EtcdPrefix,
		managementClusterResourceGroup: config.ManagementClusterResourceGroup,
		vnetMaskSize:                   config.VnetMaskSize,
	}

	return newResource, nil
}

func (r *Resource) Name() string {
	return Name
}
