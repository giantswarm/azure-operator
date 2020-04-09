// +build k8srequired

package update

import (
	"strings"

	"github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/azure-operator/integration/env"
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
	customObject, err := p.g8sClient.ProviderV1alpha1().AzureConfigs("default").Get(p.clusterID, metav1.GetOptions{})
	if err != nil {
		return v1alpha1.StatusCluster{}, microerror.Mask(err)
	}

	return customObject.Status.Cluster, nil
}

func (p *Provider) CurrentVersion() (string, error) {
	return env.VersionBundleVersion(), nil
}

func (p *Provider) NextVersion() (string, error) {
	return strings.TrimSuffix(env.VersionBundleVersion(), "-dev"), nil
}

func (p *Provider) UpdateVersion(nextVersion string) error {
	customObject, err := p.g8sClient.ProviderV1alpha1().AzureConfigs("default").Get(p.clusterID, metav1.GetOptions{})
	if err != nil {
		return microerror.Mask(err)
	}

	customObject.Spec.Cluster.Kubernetes.Kubelet.Labels = ensureLabel(customObject.Spec.Cluster.Kubernetes.Kubelet.Labels, "azure-operator.giantswarm.io/version", nextVersion)
	customObject.Spec.VersionBundle.Version = nextVersion

	_, err = p.g8sClient.ProviderV1alpha1().AzureConfigs("default").Update(customObject)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
