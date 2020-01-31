// +build k8srequired

package integration

import (
	"github.com/giantswarm/microerror"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	aggregationclient "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"

	"github.com/giantswarm/e2e-harness/pkg/harness"
)

func getK8sClient() (kubernetes.Interface, error) {
	config, err := clientcmd.BuildConfigFromFlags("", harness.DefaultKubeConfig)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	cs, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return cs, nil
}

func getK8sAggregationClient() (*aggregationclient.Clientset, error) {
	config, err := clientcmd.BuildConfigFromFlags("", harness.DefaultKubeConfig)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	acs, err := aggregationclient.NewForConfig(config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return acs, nil
}
