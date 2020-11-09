// +build k8srequired

package clusterautoscaler

import (
	"context"
	"fmt"
	"testing"

	"github.com/giantswarm/backoff"
	"github.com/giantswarm/e2e-harness/v3/pkg/framework"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	deploymentName      = "helloworld"
	deploymentNamespace = v1.NamespaceDefault
)

func Test_clusterautoscaler(t *testing.T) {
	err := autoscaler.Test(context.Background())
	if err != nil {
		t.Fatalf("%#v", err)
	}
}

type Config struct {
	Logger micrologger.Logger
	Guest  *framework.Guest
}

type ClusterAutoscaler struct {
	logger micrologger.Logger
	guest  *framework.Guest
}

func New(config Config) (*ClusterAutoscaler, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.Guest == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Guest must not be empty", config)
	}

	s := &ClusterAutoscaler{
		logger: config.Logger,
		guest:  config.Guest,
	}

	return s, nil
}

func (s *ClusterAutoscaler) Test(ctx context.Context) error {
	expectedNodes := 2
	err := s.WaitForNodesReady(ctx, expectedNodes)
	if err != nil {
		return microerror.Mask(err)
	}

	// Install deployment with expectedNodes + 1 replicas and node anti affinity.
	s.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Ensuring deployment with %d replicas", expectedNodes+1))
	err = s.ensureDeployment(ctx, expectedNodes+1)
	if err != nil {
		return microerror.Mask(err)
	}

	// Expect for expectedNodes + 1 nodes to be ready.
	err = s.WaitForNodesReady(ctx, expectedNodes+1)
	if err != nil {
		return microerror.Mask(err)
	}

	// Delete deployment.
	s.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Deleting deployment %s", deploymentName))
	err = s.deleteDeployment(ctx)
	if err != nil {
		return microerror.Mask(err)
	}
	s.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Deleted deployment %s", deploymentName))

	// Expect for expectedNodes nodes to be ready.
	err = s.WaitForNodesReady(ctx, expectedNodes)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

// The deleteDeployment function deletes the Deployment from the guest cluster.
func (s *ClusterAutoscaler) deleteDeployment(ctx context.Context) error {
	err := s.guest.K8sClient().AppsV1().Deployments(deploymentNamespace).Delete(ctx, deploymentName, metav1.DeleteOptions{})
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

// The ensureDeployment function creates a new Deployment in the guest cluster with `replicas` replicas.
func (s *ClusterAutoscaler) ensureDeployment(ctx context.Context, replicas int) error {
	replicas32 := int32(replicas)
	labelName := "app"
	labelValue := "helloworld"
	labels := map[string]string{labelName: labelValue}
	dep := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName,
			Namespace: deploymentNamespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas32,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: v1.PodSpec{
					Affinity: &v1.Affinity{
						PodAntiAffinity: &v1.PodAntiAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: []v1.PodAffinityTerm{
								{
									LabelSelector: &metav1.LabelSelector{
										MatchExpressions: []metav1.LabelSelectorRequirement{
											{
												Key:      labelName,
												Operator: metav1.LabelSelectorOpIn,
												Values:   []string{labelValue},
											},
										},
									},
									TopologyKey: "kubernetes.io/hostname",
								},
							},
						},
					},
					Containers: []v1.Container{
						{
							Name:  "helloworld",
							Image: "quay.io/giantswarm/helloworld:latest",
						},
					},
				},
			},
		},
	}
	_, err := s.guest.K8sClient().AppsV1().Deployments(deploymentNamespace).Create(ctx, &dep, metav1.CreateOptions{})
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func getWorkerNodes(ctx context.Context, k8sclient kubernetes.Interface) (*v1.NodeList, error) {
	labelSelector := fmt.Sprintf("role=%s", "worker")

	listOptions := metav1.ListOptions{
		LabelSelector: labelSelector,
	}
	nodes, err := k8sclient.CoreV1().Nodes().List(ctx, listOptions)
	if err != nil {
		return &v1.NodeList{}, microerror.Mask(err)
	}

	return nodes, nil
}

func (s *ClusterAutoscaler) WaitForNodesReady(ctx context.Context, expectedNodes int) error {
	s.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("waiting for %d k8s nodes to be in %#q state", expectedNodes, v1.NodeReady))

	o := func() error {
		nodes, err := getWorkerNodes(ctx, s.guest.K8sClient())
		if err != nil {
			return microerror.Mask(err)
		}

		var nodesReady int
		for _, n := range nodes.Items {
			for _, c := range n.Status.Conditions {
				if c.Type == v1.NodeReady && c.Status == v1.ConditionTrue {
					nodesReady++
				}
			}
		}

		if nodesReady != expectedNodes {
			return microerror.Maskf(waitError, "found %d/%d k8s nodes in %#q state but %d are expected", nodesReady, len(nodes.Items), v1.NodeReady, expectedNodes)
		}

		return nil
	}
	b := backoff.NewConstant(backoff.LongMaxWait, backoff.LongMaxInterval)
	n := backoff.NewNotifier(s.logger, ctx)

	err := backoff.RetryNotify(o, b, n)
	if err != nil {
		return microerror.Mask(err)
	}

	s.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("waited for %d k8s nodes to be in %#q state", expectedNodes, v1.NodeReady))
	return nil
}
