// +build k8srequired

package cptcconnectivity

import (
	"context"
	"testing"

	"github.com/giantswarm/apiextensions/v3/pkg/clientset/versioned"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/azure-operator/v5/e2e/env"
	"github.com/giantswarm/azure-operator/v5/e2e/setup"
)

func Test_Connectivity(t *testing.T) {

	err := connectivity.Test(context.Background())
	if err != nil {
		t.Fatalf("%#v", err)
	}
}

type Config struct {
	ClusterID string
	Logger    micrologger.Logger
	G8sClient versioned.Interface
	K8sClient kubernetes.Interface
}

type Connectivity struct {
	clusterID string
	logger    micrologger.Logger
	g8sClient versioned.Interface
	k8sClient kubernetes.Interface
}

func New(config Config) (*Connectivity, error) {
	if config.ClusterID == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.ClusterID must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.G8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.G8sClient must not be empty", config)
	}
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}

	s := &Connectivity{
		clusterID: config.ClusterID,
		logger:    config.Logger,
		g8sClient: config.G8sClient,
		k8sClient: config.K8sClient,
	}

	return s, nil
}

func (s *Connectivity) Test(ctx context.Context) error {
	s.logger.Debugf(ctx, "testing connectivity between control plane cluster and tenant cluster")
	podName := "e2e-connectivity"
	podNamespace := setup.OrganizationNamespace
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: podNamespace,
		},
		Spec: v1.PodSpec{
			RestartPolicy: v1.RestartPolicyNever,
			Containers:    []v1.Container{{Name: "connectivity", Image: "busybox", Command: []string{"nc"}, Args: []string{"-z", "api." + s.clusterID + ".k8s." + env.CommonDomain(), "443"}}},
		},
	}
	_, err := s.k8sClient.CoreV1().Pods(podNamespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		return microerror.Mask(err)
	}

	o := func() error {
		pod, err = s.k8sClient.CoreV1().Pods(podNamespace).Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			return microerror.Maskf(executionFailedError, "can't find %#q pod on control plane", podName)
		}

		if pod.Status.Phase != v1.PodSucceeded {
			return microerror.Maskf(executionFailedError, "container didn't finish yet, pod state is %#q", pod.Status.Phase)
		}

		if pod.Status.ContainerStatuses[0].State.Terminated.ExitCode != 0 {
			return microerror.Maskf(executionFailedError, "expected container exit code is 0, got %d", pod.Status.ContainerStatuses[0].LastTerminationState.Terminated.ExitCode)
		}

		return nil
	}
	b := backoff.NewExponential(backoff.ShortMaxWait, backoff.ShortMaxInterval)
	n := backoff.NewNotifier(s.logger, ctx)
	err = backoff.RetryNotify(o, b, n)
	if err != nil {
		return microerror.Maskf(executionFailedError, "couldn't connect from control plane cluster to tenant cluster")
	}

	return nil
}
