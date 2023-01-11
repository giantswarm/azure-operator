package ipam

import (
	"context"
	"net"

	"github.com/giantswarm/apiextensions/v6/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v7/service/network"
)

type AzureConfigPersisterConfig struct {
	CtrlClient client.Client
	Logger     micrologger.Logger
}

type AzureConfigPersister struct {
	ctrlClient client.Client
	logger     micrologger.Logger
}

func NewAzureConfigPersister(config AzureConfigPersisterConfig) (*AzureConfigPersister, error) {
	if config.CtrlClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CtrlClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	p := &AzureConfigPersister{
		ctrlClient: config.CtrlClient,
		logger:     config.Logger,
	}

	return p, nil
}

func (p *AzureConfigPersister) Persist(ctx context.Context, vnet net.IPNet, namespace string, name string) error {
	azureConfig := &v1alpha1.AzureConfig{}
	err := p.ctrlClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, azureConfig)
	if err != nil {
		return microerror.Mask(err)
	}

	azureNetwork, err := network.Compute(vnet)
	if err != nil {
		return microerror.Mask(err)
	}

	azureConfig.Spec.Azure.VirtualNetwork.CIDR = vnet.String()
	azureConfig.Spec.Azure.VirtualNetwork.CalicoSubnetCIDR = azureNetwork.Calico.String()
	azureConfig.Spec.Azure.VirtualNetwork.MasterSubnetCIDR = azureNetwork.Master.String()
	azureConfig.Spec.Azure.VirtualNetwork.WorkerSubnetCIDR = azureNetwork.Worker.String()

	// TODO ensure if this fields are used or can be removed.
	// NOTE in Azure we disable Calico right now. This is due to a transitioning
	// phase. The k8scloudconfig templates require certain calico valus to be set
	// nonetheless. So we set them here. Later when the Calico setup is
	// straightened out we can improve the handling here.
	azureConfig.Spec.Cluster.Calico.Subnet = azureNetwork.Calico.IP.String()
	azureConfig.Spec.Cluster.Calico.CIDR, _ = azureNetwork.Calico.Mask.Size()

	err = p.ctrlClient.Update(ctx, azureConfig)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
