// +build k8srequired

package nodepool

import (
	"context"
	"fmt"
	"testing"

	"github.com/giantswarm/apiextensions/v3/pkg/label"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/e2e-harness/v3/pkg/framework"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v5/e2e/env"
)

func Test_Nodepool(t *testing.T) {
	err := nodepool.Test(context.Background())
	if err != nil {
		t.Fatalf("%#v", err)
	}
}

type Config struct {
	ClusterID  string
	CtrlClient client.Client
	Guest      *framework.Guest
	Logger     micrologger.Logger
	Provider   *Provider
	NodePoolID string
}

type Nodepool struct {
	clusterID  string
	ctrlClient client.Client
	guest      *framework.Guest
	logger     micrologger.Logger
	provider   *Provider
	nodePoolID string
}

func New(config Config) (*Nodepool, error) {
	if config.ClusterID == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.ClusterID must not be empty", config)
	}
	if config.CtrlClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CtrlClient must not be empty", config)
	}
	if config.Guest == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Guest must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.NodePoolID == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.NodePoolID must not be empty", config)
	}
	if config.Provider == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Provider must not be empty", config)
	}

	s := &Nodepool{
		clusterID:  config.ClusterID,
		ctrlClient: config.CtrlClient,
		guest:      config.Guest,
		logger:     config.Logger,
		nodePoolID: config.NodePoolID,
		provider:   config.Provider,
	}

	return s, nil
}

const LabelVmSize = "node.kubernetes.io/instance-type"

func (s *Nodepool) Test(ctx context.Context) error {
	var err error

	clusterID := s.clusterID

	setupNodePoolID := s.nodePoolID
	setupNodePoolVmSize := env.AzureVMSize()
	setupNodePoolReplicas := 1

	newNodepoolID := "t3st"
	newNodePoolVmSize := "Standard_D3_v2"
	newNodePoolReplicas := 1

	err = s.WaitForNodesReady(ctx, newNodePoolReplicas+setupNodePoolReplicas)
	if err != nil {
		return microerror.Mask(err)
	}

	err = s.assertRightNumberOfNodes(ctx, setupNodePoolID, newNodepoolID, setupNodePoolReplicas, newNodePoolReplicas, setupNodePoolVmSize, newNodePoolVmSize)
	if err != nil {
		return microerror.Mask(err)
	}

	err = s.provider.AddWorker(ctx, clusterID, newNodepoolID)
	if err != nil {
		return microerror.Mask(err)
	}

	err = s.WaitForNodesReady(ctx, newNodePoolReplicas+setupNodePoolReplicas+1)
	if err != nil {
		return microerror.Mask(err)
	}

	err = s.assertRightNumberOfNodes(ctx, setupNodePoolID, newNodepoolID, setupNodePoolReplicas, newNodePoolReplicas+1, setupNodePoolVmSize, newNodePoolVmSize)
	if err != nil {
		return microerror.Mask(err)
	}

	err = s.provider.ChangeVmSize(ctx, clusterID, newNodepoolID, "Standard_D5_v2")
	if err != nil {
		return microerror.Mask(err)
	}

	err = s.assertRightNumberOfNodes(ctx, setupNodePoolID, newNodepoolID, setupNodePoolReplicas, newNodePoolReplicas+1, setupNodePoolVmSize, "Standard_D5_v2")
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

// assertRightNumberOfNodes Asserts that two different node pools contain different number of nodes.
// Since normally node pools will have different virtual machine sizes, I'm selecting nodes by nodepool label or by vm size label, and making sure there are the right number.
func (s *Nodepool) assertRightNumberOfNodes(ctx context.Context, setupNodePoolID, newNodePoolID string, setupNodePoolReplicas, newNodePoolReplicas int, setupNodePoolVMSize, newNodePoolVMSize string) error {
	o := func() error {
		nodes, err := getWorkerNodes(ctx, s.guest.K8sClient())
		if err != nil {
			return microerror.Mask(err)
		}

		// Check node pool created during setup.
		{
			s.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("asserting node pool %#q has %d nodes", setupNodePoolID, setupNodePoolReplicas))
			err = assertRightNumberOfNodesByNodePool(nodes, setupNodePoolID, setupNodePoolReplicas)
			if err != nil {
				return microerror.Mask(err)
			}
			s.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("asserted node pool %#q has %d nodes", setupNodePoolID, setupNodePoolReplicas))

			s.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("asserting node pool with VM Size %#q has %d nodes", setupNodePoolVMSize, setupNodePoolReplicas))
			err = assertRightNumberOfNodesByVmSize(nodes, setupNodePoolVMSize, setupNodePoolReplicas)
			if err != nil {
				return microerror.Mask(err)
			}
			s.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("asserted node pool with VM Size %#q has %d nodes", setupNodePoolVMSize, setupNodePoolReplicas))
		}

		// Check node pool created in test.
		{
			s.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("asserting node pool %#q has %d nodes", newNodePoolID, newNodePoolReplicas))
			err = assertRightNumberOfNodesByNodePool(nodes, newNodePoolID, newNodePoolReplicas)
			if err != nil {
				return microerror.Mask(err)
			}
			s.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("asserted node pool %#q has %d nodes", newNodePoolID, newNodePoolReplicas))

			s.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("asserting node pool with VM Size %#q has %d nodes", newNodePoolVMSize, newNodePoolReplicas))
			err = assertRightNumberOfNodesByVmSize(nodes, newNodePoolVMSize, newNodePoolReplicas)
			if err != nil {
				return microerror.Mask(err)
			}
			s.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("asserted node pool with VM Size %#q has %d nodes", newNodePoolVMSize, newNodePoolReplicas))
		}

		return nil
	}

	b := backoff.NewConstant(backoff.LongMaxWait, backoff.LongMaxInterval)
	n := backoff.NewNotifier(s.logger, ctx)
	err := backoff.RetryNotify(o, b, n)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func assertRightNumberOfNodesByNodePool(nodes *v1.NodeList, nodepoolID string, expectedNumberOfNodes int) error {
	existingNodesInNodePool, err := getNumberOfNodesByLabel(nodes, label.MachinePool, nodepoolID)
	if err != nil {
		return microerror.Mask(err)
	}

	if existingNodesInNodePool != expectedNumberOfNodes {
		return microerror.Maskf(unexpectedNumberOfNodesError, fmt.Sprintf("Expected %d, got %d", expectedNumberOfNodes, existingNodesInNodePool))
	}

	return nil
}

func assertRightNumberOfNodesByVmSize(nodes *v1.NodeList, vmSize string, expectedNumberOfNodes int) error {
	existingNodesInNodePool, err := getNumberOfNodesByLabel(nodes, LabelVmSize, vmSize)
	if err != nil {
		return microerror.Mask(err)
	}

	if existingNodesInNodePool != expectedNumberOfNodes {
		return microerror.Maskf(unexpectedNumberOfNodesError, fmt.Sprintf("Expected %d, got %d", expectedNumberOfNodes, existingNodesInNodePool))
	}

	return nil
}

// getNumberOfNodesByLabel returns how many nodes contain the given label with the passed value.
func getNumberOfNodesByLabel(nodes *v1.NodeList, labelName, labelValue string) (int, error) {
	existingNodes := 0

	for _, node := range nodes.Items {
		existingLabelValue, exists := node.GetLabels()[labelName]
		if !exists {
			return 0, microerror.Maskf(missingNodePoolLabelError, fmt.Sprintf("Label %#q is missing from node %#q. Present labels are %v", labelName, node.Name, node.GetLabels()))
		}

		if existingLabelValue == labelValue {
			existingNodes++
		}
	}

	return existingNodes, nil
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

func (s *Nodepool) WaitForNodesReady(ctx context.Context, expectedNodes int) error {
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
