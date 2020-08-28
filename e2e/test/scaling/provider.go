// +build k8srequired

package scaling

import (
	"context"
	"fmt"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/giantswarm/e2e-harness/pkg/framework"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/cluster-api/exp/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ProviderConfig struct {
	GuestFramework *framework.Guest
	HostFramework  *framework.Host
	Logger         micrologger.Logger

	ClusterID  string
	CtrlClient client.Client
}

type Provider struct {
	guestFramework *framework.Guest
	hostFramework  *framework.Host
	logger         micrologger.Logger

	clusterID  string
	ctrlClient client.Client
}

func NewProvider(config ProviderConfig) (*Provider, error) {
	if config.GuestFramework == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.GuestFramework must not be empty", config)
	}
	if config.HostFramework == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.HostFramework must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.ClusterID == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.ClusterID must not be empty", config)
	}
	if config.CtrlClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CtrlClient must not be empty", config)
	}

	p := &Provider{
		guestFramework: config.GuestFramework,
		hostFramework:  config.HostFramework,
		logger:         config.Logger,

		clusterID:  config.ClusterID,
		ctrlClient: config.CtrlClient,
	}

	return p, nil
}

func (p *Provider) findMachinePool(ctx context.Context) (*v1alpha3.MachinePool, error) {
	crs := &v1alpha3.MachinePoolList{}

	var labelSelector client.MatchingLabels
	{
		labelSelector = make(map[string]string)
		labelSelector[capiv1alpha3.ClusterLabelName] = p.clusterID
	}

	err := p.ctrlClient.List(ctx, crs, labelSelector, client.InNamespace(metav1.NamespaceDefault))
	if err != nil {
		return &v1alpha3.MachinePool{}, microerror.Mask(err)
	}
	if len(crs.Items) < 1 {
		p.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("MachinePool CR for cluster id %q not found", p.clusterID))
		return &v1alpha3.MachinePool{}, microerror.Maskf(notFoundError, fmt.Sprintf("MachinePool CR for cluster id %q not found", p.clusterID))
	}

	return &crs.Items[0], nil
}

func (p *Provider) AddWorker() error {
	ctx := context.Background()
	machinePool, err := p.findMachinePool(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	machinePool.Spec.Replicas = to.Int32Ptr(*machinePool.Spec.Replicas + int32(1))

	err = p.ctrlClient.Update(ctx, machinePool)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (p *Provider) NumMasters() (int, error) {
	customObject, err := p.hostFramework.G8sClient().ProviderV1alpha1().AzureConfigs(metav1.NamespaceDefault).Get(p.clusterID, metav1.GetOptions{})
	if err != nil {
		return 0, microerror.Mask(err)
	}

	num := len(customObject.Spec.Azure.Masters)

	return num, nil
}

func (p *Provider) NumWorkers() (int, error) {
	ctx := context.Background()
	machinePool, err := p.findMachinePool(ctx)
	if err != nil {
		return 0, microerror.Mask(err)
	}

	return int(*machinePool.Spec.Replicas), nil
}

func (p *Provider) RemoveWorker() error {
	ctx := context.Background()
	machinePool, err := p.findMachinePool(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	machinePool.Spec.Replicas = to.Int32Ptr(*machinePool.Spec.Replicas - int32(1))

	err = p.ctrlClient.Update(ctx, machinePool)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (p *Provider) WaitForNodes(ctx context.Context, num int) error {
	err := p.guestFramework.WaitForNodesReady(ctx, num)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
