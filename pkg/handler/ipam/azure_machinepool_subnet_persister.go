package ipam

import (
	"context"
	"fmt"
	"net"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	capzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	"sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v5/pkg/helpers"
)

type AzureMachinePoolSubnetPersisterConfig struct {
	CtrlClient client.Client
	Logger     micrologger.Logger
}

// AzureMachinePoolSubnetPersister is a Persister implementation that saves a
// subnet allocated for a node pool by adding it to AzureCluster CR.
type AzureMachinePoolSubnetPersister struct {
	ctrlClient client.Client
	logger     micrologger.Logger
}

func NewAzureMachinePoolSubnetPersister(config AzureMachinePoolSubnetPersisterConfig) (*AzureMachinePoolSubnetPersister, error) {
	if config.CtrlClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CtrlClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	p := &AzureMachinePoolSubnetPersister{
		ctrlClient: config.CtrlClient,
		logger:     config.Logger,
	}

	return p, nil
}

// Persist functions takes a subnet CIDR allocated for the specified
// AzureMachinePool (namespace/ name) and adds it to Subnets array in the
// corresponding AzureCluster CR that owns the specified AzureMachinePool.
func (p *AzureMachinePoolSubnetPersister) Persist(ctx context.Context, subnet net.IPNet, namespace string, name string) error {
	p.logger.Debugf(ctx, "persisting allocated subnet in AzureCluster CR")

	azureMachinePool := &v1alpha3.AzureMachinePool{}
	err := p.ctrlClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, azureMachinePool)
	if err != nil {
		return microerror.Mask(err)
	}

	err = p.addSubnetToAzureCluster(ctx, subnet, azureMachinePool)
	if err != nil {
		return microerror.Mask(err)
	}

	p.logger.Debugf(ctx, "persisted allocated subnet in AzureCluster CR")
	return nil
}

func (p *AzureMachinePoolSubnetPersister) addSubnetToAzureCluster(ctx context.Context, subnet net.IPNet, azureMachinePool *v1alpha3.AzureMachinePool) error {
	azureCluster, err := helpers.GetAzureClusterFromMetadata(ctx, p.ctrlClient, azureMachinePool.ObjectMeta)
	if err != nil {
		errorMessage := fmt.Sprint("error while getting AzureCluster CR from AzureMachinePool CR metadata")
		p.logger.LogCtx(ctx, "level", "warning", "message", errorMessage)
		return microerror.Mask(err)
	}

	azureMachinePoolSubnet := &capzv1alpha3.SubnetSpec{
		Role:       capzv1alpha3.SubnetNode,
		Name:       azureMachinePool.Name,
		CIDRBlocks: []string{subnet.String()},
	}
	azureCluster.Spec.NetworkSpec.Subnets = append(azureCluster.Spec.NetworkSpec.Subnets, azureMachinePoolSubnet)

	err = p.ctrlClient.Update(ctx, azureCluster)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
