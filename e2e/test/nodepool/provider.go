// +build k8srequired

package nodepool

import (
	"context"
	"fmt"

	"github.com/Azure/go-autorest/autorest/to"
	apiextensionsannotations "github.com/giantswarm/apiextensions/v2/pkg/annotation"
	"github.com/giantswarm/apiextensions/v2/pkg/label"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1alpha32 "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/cluster-api/exp/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ProviderConfig struct {
	CtrlClient client.Client
	Logger     micrologger.Logger
}

type Provider struct {
	ctrlClient client.Client
	logger     micrologger.Logger
}

func NewProvider(config ProviderConfig) (*Provider, error) {
	if config.CtrlClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CtrlClient must not be empty", config)
	}

	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	p := &Provider{
		ctrlClient: config.CtrlClient,
		logger:     config.Logger,
	}

	return p, nil
}

func (p *Provider) AddWorker(ctx context.Context, clusterID, nodepoolID string) error {
	p.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("scaling up one worker in node pool %#q", nodepoolID))

	machinePool, err := p.findMachinePool(ctx, clusterID, nodepoolID)
	if err != nil {
		return microerror.Mask(err)
	}

	newSize := *machinePool.Spec.Replicas + int32(1)
	machinePool.Spec.Replicas = to.Int32Ptr(newSize)
	machinePool.Annotations[apiextensionsannotations.NodePoolMinSize] = fmt.Sprintf("%d", newSize)
	machinePool.Annotations[apiextensionsannotations.NodePoolMaxSize] = fmt.Sprintf("%d", newSize)

	err = p.ctrlClient.Update(ctx, machinePool)
	if err != nil {
		return microerror.Mask(err)
	}

	p.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("scaled up one worker in node pool %#q", nodepoolID))

	return nil
}

func (p *Provider) ChangeVmSize(ctx context.Context, clusterID, nodepoolID, vmSize string) error {
	p.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("changing virtual machine size in node pool %#q to %s", nodepoolID, vmSize))

	machinePool, err := p.findMachinePool(ctx, clusterID, nodepoolID)
	if err != nil {
		return microerror.Mask(err)
	}

	azureMachinePool := &v1alpha32.AzureMachinePool{}
	err = p.ctrlClient.Get(ctx, client.ObjectKey{Namespace: machinePool.Namespace, Name: nodepoolID}, azureMachinePool)
	if err != nil {
		return microerror.Mask(err)
	}

	if azureMachinePool.Spec.Template.VMSize == vmSize {
		return microerror.Maskf(sameVmSizeError, "choose a different VMSize than the current one")
	}

	azureMachinePool.Spec.Template.VMSize = vmSize

	err = p.ctrlClient.Update(ctx, azureMachinePool)
	if err != nil {
		return microerror.Mask(err)
	}

	p.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("changed virtual machine size in node pool %#q to %s", nodepoolID, vmSize))

	return nil
}

func (p *Provider) findMachinePool(ctx context.Context, clusterID, nodepoolID string) (*v1alpha3.MachinePool, error) {
	crs := &v1alpha3.MachinePoolList{}

	var labelSelector client.MatchingLabels
	{
		labelSelector = make(map[string]string)
		labelSelector[capiv1alpha3.ClusterLabelName] = clusterID
		labelSelector[label.MachinePool] = nodepoolID
	}

	err := p.ctrlClient.List(ctx, crs, labelSelector, client.InNamespace(metav1.NamespaceDefault))
	if err != nil {
		return &v1alpha3.MachinePool{}, microerror.Mask(err)
	}
	if len(crs.Items) < 1 {
		p.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("MachinePool CR for cluster id %q not found", clusterID))
		return &v1alpha3.MachinePool{}, microerror.Maskf(notFoundError, fmt.Sprintf("MachinePool CR for cluster id %q not found", clusterID))
	}

	return &crs.Items[0], nil
}
