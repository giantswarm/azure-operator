package ipam

import (
	"context"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v4/pkg/helpers"
)

type AzureMachinePoolSubnetCheckerConfig struct {
	CtrlClient client.Client
	Logger     micrologger.Logger
}

// AzureMachinePoolSubnetChecker is a Checker implementation that checks if a subnet is allocated for the
// node pool specified in Check function.
type AzureMachinePoolSubnetChecker struct {
	ctrlClient client.Client
	logger     micrologger.Logger
}

func NewAzureMachinePoolSubnetChecker(config AzureMachinePoolSubnetCheckerConfig) (*AzureMachinePoolSubnetChecker, error) {
	if config.CtrlClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CtrlClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	c := &AzureMachinePoolSubnetChecker{
		ctrlClient: config.CtrlClient,
		logger:     config.Logger,
	}

	return c, nil
}

// Check function checks if a subnet is allocated for the specified AzureMachinePool. It is
// checking if the allocated subnet is set in the corresponding Cluster CR that owns specified
// AzureMachinePool.
func (c *AzureMachinePoolSubnetChecker) Check(ctx context.Context, namespace string, name string) (bool, error) {
	c.logger.LogCtx(ctx, "level", "debug", "message", "checking if node pool subnet has to be allocated")

	objectKey := client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}
	azureMachinePool := &v1alpha3.AzureMachinePool{}
	err := c.ctrlClient.Get(ctx, objectKey, azureMachinePool)
	if err != nil {
		return false, microerror.Mask(err)
	}

	azureCluster, err := helpers.GetAzureClusterFromMetadata(ctx, c.ctrlClient, azureMachinePool.ObjectMeta)
	if err != nil {
		return false, microerror.Mask(err)
	}

	// In case there is no subnet tracked so far, we want to proceed with the allocation process.
	for _, subnet := range azureCluster.Spec.NetworkSpec.Subnets {
		if subnet.Name == azureMachinePool.Name {
			c.logger.LogCtx(ctx, "level", "debug", "message", "found existing node pool subnet")
			return false, nil
		}
	}

	c.logger.LogCtx(ctx, "level", "debug", "message", "node pool subnet not found, new subnet has to be allocated")
	return true, nil
}
