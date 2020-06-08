package ipam

import (
	"context"
	"net"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	capzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	"sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v4/pkg/annotation"
)

type AzureMachinePoolPersisterConfig struct {
	CtrlClient client.Client
	Logger     micrologger.Logger
}

type AzureMachinePoolPersister struct {
	ctrlClient client.Client
	logger     micrologger.Logger
}

func NewAzureMachinePoolPersister(config AzureMachinePoolPersisterConfig) (*AzureMachinePoolPersister, error) {
	if config.CtrlClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.G8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	p := &AzureMachinePoolPersister{
		ctrlClient: config.CtrlClient,
		logger:     config.Logger,
	}

	return p, nil
}

func (p *AzureMachinePoolPersister) Persist(ctx context.Context, vnet net.IPNet, namespace string, name string) error {
	{
		azureCluster := &capzv1alpha3.AzureCluster{}
		err := p.ctrlClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, azureCluster)
		if err != nil {
			return microerror.Mask(err)
		}

		azureMachinePoolSubnet := &capzv1alpha3.SubnetSpec{
			Role:      capzv1alpha3.SubnetNode,
			Name:      name,
			CidrBlock: vnet.String(),
		}
		azureCluster.Spec.NetworkSpec.Subnets = append(azureCluster.Spec.NetworkSpec.Subnets, azureMachinePoolSubnet)

		err = p.ctrlClient.Update(ctx, azureCluster)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	{
		azureMachinePool := &v1alpha3.AzureMachinePool{}
		err := p.ctrlClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, azureMachinePool)
		if err != nil {
			return microerror.Mask(err)
		}

		if azureMachinePool.GetAnnotations() == nil {
			azureMachinePool.Annotations = map[string]string{}
		}

		azureMachinePool.GetAnnotations()[annotation.AzureMachinePoolSubnet] = vnet.String()

		err = p.ctrlClient.Update(ctx, azureMachinePool)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}
