package ipam

import (
	"context"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v4/pkg/annotation"
)

type AzureMachinePoolCheckerConfig struct {
	CtrlClient client.Client
	Logger     micrologger.Logger
}

type AzureMachinePoolChecker struct {
	ctrlClient client.Client
	logger     micrologger.Logger
}

func NewAzureMachinePoolChecker(config AzureMachinePoolCheckerConfig) (*AzureMachinePoolChecker, error) {
	if config.CtrlClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CtrlClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	a := &AzureMachinePoolChecker{
		ctrlClient: config.CtrlClient,
		logger:     config.Logger,
	}

	return a, nil
}

func (c *AzureMachinePoolChecker) Check(ctx context.Context, namespace string, name string) (bool, error) {
	azureMachinePool := &v1alpha3.AzureMachinePool{}
	err := c.ctrlClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, azureMachinePool)
	if err != nil {
		return false, microerror.Mask(err)
	}

	// In case there is no subnet tracked so far, we want to proceed with the allocation process.
	_, alreadyAllocated := azureMachinePool.GetAnnotations()[annotation.AzureMachinePoolSubnet]
	return !alreadyAllocated, nil
}
