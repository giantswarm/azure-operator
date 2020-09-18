package ipam

import (
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/azure-operator/v4/pkg/locker"
)

const (
	Name                                 = "ipam"
	SubnetRange         NetworkRangeType = "subnet"
	VirtualNetworkRange NetworkRangeType = "virtual network"
)

type NetworkRangeType string

type Config struct {
	Checker            Checker
	Collector          Collector
	Locker             locker.Interface
	Logger             micrologger.Logger
	NetworkRangeGetter NetworkRangeGetter
	NetworkRangeType   NetworkRangeType
	Persister          Persister
}

// Resource finds free IP ranges:
// - AzureConfig: within an installation range to create new virtual network for the tenant cluster.
// - CAPI/CAPZ: within a virtual network to create new subnets.
type Resource struct {
	checker            Checker
	collector          Collector
	locker             locker.Interface
	logger             micrologger.Logger
	networkRangeGetter NetworkRangeGetter
	networkRangeType   NetworkRangeType
	persister          Persister
}

func New(config Config) (*Resource, error) {
	if config.Checker == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Checker must not be empty", config)
	}
	if config.Collector == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Collector must not be empty", config)
	}
	if config.Locker == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Locker must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.NetworkRangeGetter == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.NetworkRangeGetter must not be empty", config)
	}
	if config.NetworkRangeType == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.NetworkRangeType must not be empty", config)
	}
	if config.Persister == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Persister must not be empty", config)
	}

	r := &Resource{
		checker:            config.Checker,
		collector:          config.Collector,
		locker:             config.Locker,
		logger:             config.Logger,
		networkRangeGetter: config.NetworkRangeGetter,
		networkRangeType:   config.NetworkRangeType,
		persister:          config.Persister,
	}

	return r, nil
}

func (r *Resource) Name() string {
	return Name
}
