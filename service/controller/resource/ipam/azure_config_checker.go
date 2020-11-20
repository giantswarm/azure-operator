package ipam

import (
	"context"
	"net"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

type AzureConfigCheckerConfig struct {
	CtrlClient client.Client
	Logger     micrologger.Logger
}

type AzureConfigChecker struct {
	ctrlClient client.Client
	logger     micrologger.Logger
}

func NewAzureConfigChecker(config AzureConfigCheckerConfig) (*AzureConfigChecker, error) {
	if config.CtrlClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CtrlClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	a := &AzureConfigChecker{
		ctrlClient: config.CtrlClient,
		logger:     config.Logger,
	}

	return a, nil
}

func (c *AzureConfigChecker) Check(ctx context.Context, namespace string, name string) (*net.IPNet, error) {
	azureCluster := &v1alpha1.AzureConfig{}
	err := c.ctrlClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, azureCluster)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// We check the subnet we want to ensure in the CR status. In case there is no
	// subnet tracked so far, we want to proceed with the allocation process. Thus
	// we return nil.
	if key.AzureConfigNetworkCIDR(*azureCluster) == "" {
		return nil, nil
	}

	_, subnet, err := net.ParseCIDR(key.AzureConfigNetworkCIDR(*azureCluster))
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return subnet, nil
}
