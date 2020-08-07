package ipam

import (
	"context"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	capzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	"sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type AzureMachinePoolCheckerConfig struct {
	CtrlClient client.Client
	Logger     micrologger.Logger
}

// AzureMachinePoolChecker is a Checker implementation that checks if a subnet is allocated for the
// node pool specified in Check function.
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

// Check function checks if a subnet is allocated for the specified AzureMachinePool. It is
// checking if the allocated subnet is set in the corresponding Cluster CR that owns specified
// AzureMachinePool.
func (c *AzureMachinePoolChecker) Check(ctx context.Context, namespace string, name string) (bool, error) {
	azureMachinePool := &v1alpha3.AzureMachinePool{}
	err := c.ctrlClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, azureMachinePool)
	if err != nil {
		return false, microerror.Mask(err)
	}

	cluster, err := util.GetClusterFromMetadata(ctx, c.ctrlClient, azureMachinePool.ObjectMeta)
	if err != nil {
		return false, microerror.Mask(err)
	}

	azureCluster, err := c.getAzureClusterFromCluster(ctx, cluster)
	if err != nil {
		return false, microerror.Mask(err)
	}

	// In case there is no subnet tracked so far, we want to proceed with the allocation process.
	for _, subnet := range azureCluster.Spec.NetworkSpec.Subnets {
		if subnet.Name == azureMachinePool.Name {
			return false, nil
		}
	}

	return true, nil
}

func (c *AzureMachinePoolChecker) getAzureClusterFromCluster(ctx context.Context, cluster *capiv1alpha3.Cluster) (*capzv1alpha3.AzureCluster, error) {
	azureCluster := &capzv1alpha3.AzureCluster{}
	azureClusterName := client.ObjectKey{
		Namespace: cluster.Spec.InfrastructureRef.Namespace,
		Name:      cluster.Spec.InfrastructureRef.Name,
	}
	err := c.ctrlClient.Get(ctx, azureClusterName, azureCluster)
	if err != nil {
		return azureCluster, microerror.Mask(err)
	}

	return azureCluster, nil
}
