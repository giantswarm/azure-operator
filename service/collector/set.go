package collector

import (
	"github.com/giantswarm/exporterkit/collector"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
)

type SetConfig struct {
	K8sClient kubernetes.Interface
	Logger    micrologger.Logger
	Watcher   func(opts metav1.ListOptions) (watch.Interface, error)

	// EnvironmentName is the name of the Azure environment used to compute the
	// azure.Environment type. See also
	// https://godoc.org/github.com/Azure/go-autorest/autorest/azure#Environment.
	EnvironmentName string
}

// Set is basically only a wrapper for the operator's collector implementations.
// It eases the iniitialization and prevents some weird import mess so we do not
// have to alias packages. There is also the benefit of the helper type kept
// private so we do not need to expose this magic.
type Set struct {
	*collector.Set
}

func NewSet(config SetConfig) (*Set, error) {
	var err error

	var deploymentCollector *Deployment
	{
		c := DeploymentConfig{
			K8sClient: config.K8sClient,
			Logger:    config.Logger,
			Watcher:   config.Watcher,

			EnvironmentName: config.EnvironmentName,
		}

		deploymentCollector, err = NewDeployment(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var collectorSet *collector.Set
	{
		c := collector.SetConfig{
			Collectors: []collector.Interface{
				deploymentCollector,
			},
			Logger: config.Logger,
		}

		collectorSet, err = collector.NewSet(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	s := &Set{
		Set: collectorSet,
	}

	return s, nil
}
