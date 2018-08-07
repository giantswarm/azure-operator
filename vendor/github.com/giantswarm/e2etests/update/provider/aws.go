package provider

import (
	"encoding/json"

	"github.com/giantswarm/e2e-harness/pkg/framework"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type AWSConfig struct {
	HostFramework *framework.Host
	Logger        micrologger.Logger

	ClusterID   string
	GithubToken string
}

type AWS struct {
	hostFramework *framework.Host
	logger        micrologger.Logger

	clusterID   string
	githubToken string
}

func NewAWS(config AWSConfig) (*AWS, error) {
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

	a := &AWS{
		hostFramework: config.HostFramework,
		logger:        config.Logger,

		clusterID:   config.ClusterID,
		githubToken: config.GithubToken,
	}

	return a, nil
}

func (a *AWS) CurrentVersion() (string, error) {
	p := &framework.VBVParams{
		Component: "aws-operator",
		Provider:  "aws",
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

func (a *AWS) IsUpdated() (bool, error) {
	customObject, err := a.hostFramework.G8sClient().ProviderV1alpha1().AWSConfigs("default").Get(a.clusterID, metav1.GetOptions{})
	if err != nil {
		return false, microerror.Mask(err)
	}

	return customObject.Status.Cluster.HasUpdatedCondition(), nil
}

func (a *AWS) NextVersion() (string, error) {
	p := &framework.VBVParams{
		Component: "aws-operator",
		Provider:  "aws",
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

func (a *AWS) UpdateVersion(nextVersion string) error {
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

	_, err = a.hostFramework.G8sClient().ProviderV1alpha1().AWSConfigs("default").Patch(a.clusterID, types.JSONPatchType, b)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
