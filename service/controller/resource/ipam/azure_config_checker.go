package ipam

import (
	"context"

	"github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v4/service/controller/key"
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

func (c *AzureConfigChecker) Check(ctx context.Context, namespace string, name string) (bool, error) {
	azureCluster := &v1alpha1.AzureConfig{}
	err := c.ctrlClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, azureCluster)
	if err != nil {
		return false, microerror.Mask(err)
	}

	// We check the subnet we want to ensure in the CR status. In case there is no
	// subnet tracked so far, we want to proceed with the allocation process. Thus
	// we return true.
	if key.AzureConfigNetworkCIDR(*azureCluster) == "" {
		return true, nil
	}

	// At this point the subnet is already allocated for the CR we check here. So
	// we do not want to proceed further and return false.
	return false, nil
}
