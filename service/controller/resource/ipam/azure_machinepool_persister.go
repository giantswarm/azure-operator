package ipam

import (
	"context"
	"net"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	capzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	"sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v4/pkg/annotation"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
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

func (p *AzureMachinePoolPersister) Persist(ctx context.Context, vnet net.IPNet, obj interface{}) error {
	azureMachinePool, err := key.ToAzureMachinePool(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	err = p.addSubnetToAzureCluster(ctx, vnet, azureMachinePool)
	if err != nil {
		return microerror.Mask(err)
	}

	err = p.addSubnetToAzureMachinePool(ctx, vnet, azureMachinePool)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (p *AzureMachinePoolPersister) addSubnetToAzureMachinePool(ctx context.Context, vnet net.IPNet, azureMachinePool v1alpha3.AzureMachinePool) error {
	if azureMachinePool.GetAnnotations() == nil {
		azureMachinePool.Annotations = map[string]string{}
	}

	azureMachinePool.GetAnnotations()[annotation.AzureMachinePoolSubnet] = vnet.String()

	err := p.ctrlClient.Update(ctx, &azureMachinePool)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (p *AzureMachinePoolPersister) addSubnetToAzureCluster(ctx context.Context, vnet net.IPNet, azureMachinePool v1alpha3.AzureMachinePool) error {
	cluster, err := util.GetClusterFromMetadata(ctx, p.ctrlClient, azureMachinePool.ObjectMeta)
	if err != nil {
		return microerror.Mask(err)
	}

	azureCluster, err := p.getAzureClusterFromCluster(ctx, cluster)
	if err != nil {
		return microerror.Mask(err)
	}

	azureMachinePoolSubnet := &capzv1alpha3.SubnetSpec{
		Role:      capzv1alpha3.SubnetNode,
		Name:      azureMachinePool.Name,
		CidrBlock: vnet.String(),
	}
	azureCluster.Spec.NetworkSpec.Subnets = append(azureCluster.Spec.NetworkSpec.Subnets, azureMachinePoolSubnet)

	err = p.ctrlClient.Update(ctx, azureCluster)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (p *AzureMachinePoolPersister) getAzureClusterFromCluster(ctx context.Context, cluster *capiv1alpha3.Cluster) (*capzv1alpha3.AzureCluster, error) {
	azureCluster := &capzv1alpha3.AzureCluster{}
	azureClusterName := client.ObjectKey{
		Namespace: cluster.Spec.InfrastructureRef.Namespace,
		Name:      cluster.Spec.InfrastructureRef.Name,
	}
	err := p.ctrlClient.Get(ctx, azureClusterName, azureCluster)
	if err != nil {
		return azureCluster, microerror.Mask(err)
	}

	return azureCluster, nil
}
