package ipam

import (
	"context"
	"net"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	capzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	expcapzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v5/pkg/helpers"
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
		return nil, microerror.Maskf(invalidConfigError, "%T.WCClients must not be empty", config)
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
// AzureMachinePool. Returns allocated subnet or nil.
func (c *AzureMachinePoolSubnetChecker) Check(ctx context.Context, namespace string, name string) (*net.IPNet, error) {
	c.logger.Debugf(ctx, "checking if node pool subnet has to be allocated")
	var err error

	var azureMachinePool *expcapzv1alpha3.AzureMachinePool
	{
		objectKey := client.ObjectKey{
			Namespace: namespace,
			Name:      name,
		}

		azureMachinePool = &expcapzv1alpha3.AzureMachinePool{}
		err = c.ctrlClient.Get(ctx, objectKey, azureMachinePool)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var azureCluster *capzv1alpha3.AzureCluster
	{
		azureCluster, err = helpers.GetAzureClusterFromMetadata(ctx, c.ctrlClient, azureMachinePool.ObjectMeta)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	// In case there is no subnet tracked so far, we want to proceed with the allocation process.
	for _, subnet := range azureCluster.Spec.NetworkSpec.Subnets {
		if subnet.Name == azureMachinePool.Name && len(subnet.CIDRBlocks) > 0 {
			_, subnet, err := net.ParseCIDR(subnet.CIDRBlocks[0])
			if err != nil {
				return nil, microerror.Mask(err)
			}

			c.logger.Debugf(ctx, "found existing node pool subnet")
			return subnet, nil
		}
	}

	c.logger.Debugf(ctx, "node pool subnet not found, new subnet has to be allocated")
	return nil, nil
}
