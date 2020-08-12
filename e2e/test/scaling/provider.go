// +build k8srequired

package scaling

import (
	"context"
	"encoding/json"

	"github.com/giantswarm/e2e-harness/v2/pkg/framework"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type ProviderConfig struct {
	GuestFramework *framework.Guest
	HostFramework  *framework.Host
	Logger         micrologger.Logger

	ClusterID string
}

type Provider struct {
	guestFramework *framework.Guest
	hostFramework  *framework.Host
	logger         micrologger.Logger

	clusterID string
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

	p := &Provider{
		guestFramework: config.GuestFramework,
		hostFramework:  config.HostFramework,
		logger:         config.Logger,

		clusterID: config.ClusterID,
	}

	return p, nil
}

func (p *Provider) AddWorker(ctx context.Context) error {
	customObject, err := p.hostFramework.G8sClient().ProviderV1alpha1().AzureConfigs("default").Get(ctx, p.clusterID, metav1.GetOptions{})
	if err != nil {
		return microerror.Mask(err)
	}

	patches := []Patch{
		{
			Op:    "add",
			Path:  "/spec/azure/workers/-",
			Value: customObject.Spec.Azure.Workers[0],
		},
	}

	b, err := json.Marshal(patches)
	if err != nil {
		return microerror.Mask(err)
	}

	_, err = p.hostFramework.G8sClient().ProviderV1alpha1().AzureConfigs("default").Patch(ctx, p.clusterID, types.JSONPatchType, b, metav1.PatchOptions{})
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (p *Provider) NumMasters(ctx context.Context) (int, error) {
	customObject, err := p.hostFramework.G8sClient().ProviderV1alpha1().AzureConfigs("default").Get(ctx, p.clusterID, metav1.GetOptions{})
	if err != nil {
		return 0, microerror.Mask(err)
	}

	num := len(customObject.Spec.Azure.Masters)

	return num, nil
}

func (p *Provider) NumWorkers(ctx context.Context) (int, error) {
	customObject, err := p.hostFramework.G8sClient().ProviderV1alpha1().AzureConfigs("default").Get(ctx, p.clusterID, metav1.GetOptions{})
	if err != nil {
		return 0, microerror.Mask(err)
	}

	num := len(customObject.Spec.Azure.Workers)

	return num, nil
}

func (p *Provider) RemoveWorker(ctx context.Context) error {
	patches := []Patch{
		{
			Op:   "remove",
			Path: "/spec/azure/workers/1",
		},
	}

	b, err := json.Marshal(patches)
	if err != nil {
		return microerror.Mask(err)
	}

	_, err = p.hostFramework.G8sClient().ProviderV1alpha1().AzureConfigs("default").Patch(ctx, p.clusterID, types.JSONPatchType, b, metav1.PatchOptions{})
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
