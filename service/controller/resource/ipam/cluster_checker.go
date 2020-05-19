package ipam

import (
	"context"

	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

type ClusterCheckerConfig struct {
	G8sClient versioned.Interface
	Logger    micrologger.Logger
}

type ClusterChecker struct {
	g8sClient versioned.Interface
	logger    micrologger.Logger
}

func NewClusterChecker(config ClusterCheckerConfig) (*ClusterChecker, error) {
	if config.G8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.G8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	a := &ClusterChecker{
		g8sClient: config.G8sClient,
		logger:    config.Logger,
	}

	return a, nil
}

func (c *ClusterChecker) Check(ctx context.Context, namespace string, name string) (bool, error) {
	cr, err := c.g8sClient.ProviderV1alpha1().AzureConfigs(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return false, microerror.Mask(err)
	}

	// We check the subnet we want to ensure in the CR status. In case there is no
	// subnet tracked so far, we want to proceed with the allocation process. Thus
	// we return true.
	if key.AzureConfigNetworkCIDR(*cr) == "" {
		return true, nil
	}

	// At this point the subnet is already allocated for the CR we check here. So
	// we do not want to proceed further and return false.
	return false, nil
}
