package provider

import (
	"encoding/json"

	"github.com/giantswarm/e2e-harness/pkg/framework"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type KVMConfig struct {
	HostFramework *framework.Host
	Logger        micrologger.Logger

	ClusterID   string
	GithubToken string
}

type KVM struct {
	hostFramework *framework.Host
	logger        micrologger.Logger

	clusterID   string
	githubToken string
}

func NewKVM(config KVMConfig) (*KVM, error) {
	if config.HostFramework == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.HostFramework must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if config.ClusterID == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.ClusterID must not be empty", config)
	}
	if config.GithubToken == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.GithubToken must not be empty", config)
	}

	k := &KVM{
		hostFramework: config.HostFramework,
		logger:        config.Logger,

		clusterID:   config.ClusterID,
		githubToken: config.GithubToken,
	}

	return k, nil
}

func (a *KVM) CurrentVersion() (string, error) {
	p := &framework.VBVParams{
		Component: "kvm-operator",
		Provider:  "kvm",
		Token:     a.githubToken,
		VType:     "current",
	}
	v, err := framework.GetVersionBundleVersion(p)
	if err != nil {
		return "", microerror.Mask(err)
	}

	if v == "" {
		return "", microerror.Mask(versionNotFoundError)
	}

	return v, nil
}

func (k *KVM) IsUpdated() (bool, error) {
	customObject, err := k.hostFramework.G8sClient().ProviderV1alpha1().KVMConfigs("default").Get(k.clusterID, metav1.GetOptions{})
	if err != nil {
		return false, microerror.Mask(err)
	}

	return customObject.Status.Cluster.HasUpdatedCondition(), nil
}

func (a *KVM) NextVersion() (string, error) {
	p := &framework.VBVParams{
		Component: "kvm-operator",
		Provider:  "kvm",
		Token:     a.githubToken,
		VType:     "wip",
	}
	v, err := framework.GetVersionBundleVersion(p)
	if err != nil {
		return "", microerror.Mask(err)
	}

	if v == "" {
		return "", microerror.Mask(versionNotFoundError)
	}

	return v, nil
}

func (k *KVM) UpdateVersion(nextVersion string) error {
	patches := []Patch{
		{
			Op:    "replace",
			Path:  "/spec/versionBundle/version",
			Value: nextVersion,
		},
	}

	b, err := json.Marshal(patches)
	if err != nil {
		return microerror.Mask(err)
	}

	_, err = k.hostFramework.G8sClient().ProviderV1alpha1().KVMConfigs("default").Patch(k.clusterID, types.JSONPatchType, b)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
