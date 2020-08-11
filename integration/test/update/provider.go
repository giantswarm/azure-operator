// +build k8srequired

package update

import (
	"context"

	"github.com/giantswarm/apiextensions/v2/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/apiextensions/v2/pkg/clientset/versioned"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/azure-operator/v4/integration/env"
)

type ProviderConfig struct {
	G8sClient versioned.Interface
	Logger    micrologger.Logger

	ClusterID string
}

type Provider struct {
	g8sClient versioned.Interface
	logger    micrologger.Logger

	clusterID string
}

func NewProvider(config ProviderConfig) (*Provider, error) {
	if config.G8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.G8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if config.ClusterID == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.ClusterID must not be empty", config)
	}

	p := &Provider{
		g8sClient: config.G8sClient,
		logger:    config.Logger,

		clusterID: config.ClusterID,
	}

	return p, nil
}

func (p *Provider) CurrentStatus() (v1alpha1.StatusCluster, error) {
	ctx := context.Background()

	customObject, err := p.g8sClient.ProviderV1alpha1().AzureConfigs("default").Get(ctx, p.clusterID, metav1.GetOptions{})
	if err != nil {
		return v1alpha1.StatusCluster{}, microerror.Mask(err)
	}

	return customObject.Status.Cluster, nil
}

func (p *Provider) CurrentVersion() (string, error) {
	return env.GetLatestOperatorRelease(), nil
}

// NextVersion returns the version which we are going to upgrade to
func (p *Provider) NextVersion() (string, error) {
	return env.VersionBundleVersion(), nil
}

func (p *Provider) UpdateVersion(nextVersion string) error {
	ctx := context.Background()

	customObject, err := p.g8sClient.ProviderV1alpha1().AzureConfigs("default").Get(ctx, p.clusterID, metav1.GetOptions{})
	if err != nil {
		return microerror.Mask(err)
	}

	customObject.Spec.Cluster.Kubernetes.Kubelet.Labels = ensureLabel(customObject.Spec.Cluster.Kubernetes.Kubelet.Labels, "azure-operator.giantswarm.io/version", nextVersion)
	customObject.Spec.VersionBundle.Version = nextVersion

	labels := customObject.GetLabels()
	labels["azure-operator.giantswarm.io/version"] = nextVersion
	customObject.SetLabels(labels)

	_, err = p.g8sClient.ProviderV1alpha1().AzureConfigs("default").Update(ctx, customObject, metav1.UpdateOptions{})
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
