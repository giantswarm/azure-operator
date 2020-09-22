// +build k8srequired

package scaling

import (
	"context"
	"fmt"

	"github.com/Azure/go-autorest/autorest/to"
	apiextensionsannotations "github.com/giantswarm/apiextensions/v2/pkg/annotation"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/e2e-harness/v2/pkg/framework"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	v1 "k8s.io/api/core/v1"
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

func (p *Provider) findMachinePools(ctx context.Context) ([]v1alpha3.MachinePool, error) {
	crs := &v1alpha3.MachinePoolList{}

	var labelSelector client.MatchingLabels
	{
		labelSelector = make(map[string]string)
		labelSelector[capiv1alpha3.ClusterLabelName] = p.clusterID
	}

	err := p.ctrlClient.List(ctx, crs, labelSelector, client.InNamespace(metav1.NamespaceDefault))
	if err != nil {
		return []v1alpha3.MachinePool{}, microerror.Mask(err)
	}

	return crs.Items, nil
}

func (p *Provider) AddWorker(ctx context.Context) error {
	machinePools, err := p.findMachinePools(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	if len(machinePools) < 1 {
		p.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("MachinePool CR for cluster id %q not found", p.clusterID))
		return microerror.Maskf(notFoundError, fmt.Sprintf("MachinePool CR for cluster id %q not found", p.clusterID))
	}

	machinePool := machinePools[0]

	newSize := *machinePool.Spec.Replicas + int32(1)
	machinePool.Spec.Replicas = to.Int32Ptr(newSize)
	machinePool.Annotations[apiextensionsannotations.NodePoolMinSize] = fmt.Sprintf("%d", newSize)
	machinePool.Annotations[apiextensionsannotations.NodePoolMaxSize] = fmt.Sprintf("%d", newSize)

	err = p.ctrlClient.Update(ctx, &machinePool)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (p *Provider) NumMasters(ctx context.Context) (int, error) {
	customObject, err := p.hostFramework.G8sClient().ProviderV1alpha1().AzureConfigs(metav1.NamespaceDefault).Get(ctx, p.clusterID, metav1.GetOptions{})
	if err != nil {
		return 0, microerror.Mask(err)
	}

	num := len(customObject.Spec.Azure.Masters)

	return num, nil
}

func (p *Provider) NumWorkers(ctx context.Context) (int, error) {
	machinePools, err := p.findMachinePools(ctx)
	if err != nil {
		return 0, microerror.Mask(err)
	}

	count := 0
	for _, mp := range machinePools {
		count = count + int(*mp.Spec.Replicas)
	}

	return count, nil
}

func (p *Provider) RemoveWorker(ctx context.Context) error {
	machinePools, err := p.findMachinePools(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	if len(machinePools) < 1 {
		p.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("MachinePool CR for cluster id %q not found", p.clusterID))
		return microerror.Maskf(notFoundError, fmt.Sprintf("MachinePool CR for cluster id %q not found", p.clusterID))
	}

	machinePool := machinePools[0]

	newSize := *machinePool.Spec.Replicas - int32(1)
	machinePool.Spec.Replicas = to.Int32Ptr(newSize)
	machinePool.Annotations[apiextensionsannotations.NodePoolMinSize] = fmt.Sprintf("%d", newSize)
	machinePool.Annotations[apiextensionsannotations.NodePoolMaxSize] = fmt.Sprintf("%d", newSize)

	err = p.ctrlClient.Update(ctx, &machinePool)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (p *Provider) WaitForNodes(ctx context.Context, expectedNodes int) error {
	p.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("waiting for %d k8s nodes to be in %#q state", expectedNodes, v1.NodeReady))

	o := func() error {
		// Get all all nodes from the kubernetes API.
		var nodesReady int
		var allNodes int
		{
			nodeList, err := p.guestFramework.K8sClient().CoreV1().Nodes().List(ctx, metav1.ListOptions{})
			if err != nil {
				return microerror.Mask(err)
			}

			for _, n := range nodeList.Items {
				for _, c := range n.Status.Conditions {
					allNodes++
					if c.Type == v1.NodeReady && c.Status == v1.ConditionTrue {
						nodesReady++
					}
				}
			}
		}

		if nodesReady != expectedNodes {
			return microerror.Maskf(waitError, "found %d/%d k8s nodes in %#q state but %d are expected", nodesReady, allNodes, v1.NodeReady, expectedNodes)
		}

		return nil
	}
	b := backoff.NewConstant(backoff.LongMaxWait, backoff.LongMaxInterval)
	n := backoff.NewNotifier(p.logger, ctx)

	err := backoff.RetryNotify(o, b, n)
	if err != nil {
		return microerror.Mask(err)
	}

	p.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("waited for %d k8s nodes to be in %#q state", expectedNodes, v1.NodeReady))
	return nil
}
