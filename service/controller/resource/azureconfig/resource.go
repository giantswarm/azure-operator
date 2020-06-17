package azureconfig

import (
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
	ClusterIPRange                 string
	EtcdPrefix                     string
	ManagementClusterResourceGroup string
	SSHUserList                    string
}

type Resource struct {
	ctrlClient client.Client
	logger     micrologger.Logger

	apiServerSecurePort            int
	calico                         CalicoConfig
	clusterIPRange                 string
	etcdPrefix                     string
	managementClusterResourceGroup string
	sshUserList                    string
}

func New(config Config) (*Resource, error) {
	if config.CtrlClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CtrlClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	// No validation for configuration at this point. I'm not fully sure if any
	// of that is actually needed.

	newResource := &Resource{
		ctrlClient: config.CtrlClient,
		logger:     config.Logger,

		apiServerSecurePort:            config.APIServerSecurePort,
		calico:                         config.Calico,
		clusterIPRange:                 config.ClusterIPRange,
		etcdPrefix:                     config.EtcdPrefix,
		managementClusterResourceGroup: config.ManagementClusterResourceGroup,
		sshUserList:                    config.SSHUserList,
	}

	return newResource, nil
}

func (r *Resource) Name() string {
	return Name
}
