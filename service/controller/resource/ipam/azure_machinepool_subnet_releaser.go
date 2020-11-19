package ipam

import (
	"context"
	"fmt"
	"net"

	"github.com/giantswarm/azure-operator/v5/pkg/helpers"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type AzureMachinePoolSubnetReleaserConfig struct {
	CtrlClient client.Client
	Logger     micrologger.Logger
}

// AzureMachinePoolSubnetReleaser is a Releaser implementation that releases an
// allocated subnet for a node pool by removing it from AzureCluster CR.
type AzureMachinePoolSubnetReleaser struct {
	ctrlClient client.Client
	logger     micrologger.Logger
}

func NewAzureMachinePoolSubnetReleaser(config AzureMachinePoolSubnetReleaserConfig) (*AzureMachinePoolSubnetReleaser, error) {
	if config.CtrlClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CtrlClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	p := &AzureMachinePoolSubnetReleaser{
		ctrlClient: config.CtrlClient,
		logger:     config.Logger,
	}

	return p, nil
}

func (r *AzureMachinePoolSubnetReleaser) Release(ctx context.Context, subnet net.IPNet, namespace, name string) error {
	r.logger.LogCtx(ctx, "level", "debug", "message", "releasing allocated subnet from AzureCluster CR")

	azureMachinePool := &v1alpha3.AzureMachinePool{}
	err := r.ctrlClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, azureMachinePool)
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.removeSubnetFromAzureCluster(ctx, subnet, azureMachinePool)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "released allocated subnet from AzureCluster CR")
	return nil
}

func (r *AzureMachinePoolSubnetReleaser) removeSubnetFromAzureCluster(ctx context.Context, subnet net.IPNet, azureMachinePool *v1alpha3.AzureMachinePool) error {
	azureCluster, err := helpers.GetAzureClusterFromMetadata(ctx, r.ctrlClient, azureMachinePool.ObjectMeta)
	if err != nil {
		errorMessage := fmt.Sprintf("error while getting AzureCluster CR from AzureMachinePool CR metadata")
		r.logger.LogCtx(ctx, "level", "warning", "message", errorMessage)
		return microerror.Mask(err)
	}

	for i, subnet := range azureCluster.Spec.NetworkSpec.Subnets {
		if subnet.Name == azureMachinePool.Name {
			azureCluster.Spec.NetworkSpec.Subnets = append(azureCluster.Spec.NetworkSpec.Subnets[:i], azureCluster.Spec.NetworkSpec.Subnets[i+1:]...)
		}
	}

	err = r.ctrlClient.Update(ctx, azureCluster)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
