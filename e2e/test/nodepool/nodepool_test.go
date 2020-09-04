// +build k8srequired

package nodepool

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	releasev1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/release/v1alpha1"
	"github.com/giantswarm/apiextensions/pkg/label"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/e2e-harness/pkg/framework"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/reference"
	expcapzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	expcapiv1alpha3 "sigs.k8s.io/cluster-api/exp/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v4/e2e/env"
	"github.com/giantswarm/azure-operator/v4/e2e/setup"
	key2 "github.com/giantswarm/azure-operator/v4/service/controller/key"
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
	clusterID := s.clusterID

	setupNodePoolID := s.nodePoolID
	setupNodePoolVmSize := env.AzureVMSize()
	setupNodePoolReplicas := 2

	newNodepoolID := "t3st"
	newNodePoolVmSize := "Standard_D3_v2"
	newNodePoolReplicas := 3

	err := s.CreateNodePool(ctx, newNodepoolID, int32(newNodePoolReplicas), newNodePoolVmSize)
	if err != nil {
		return microerror.Mask(err)
	}

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
		err = assertRightNumberOfNodesByNodePool(nodes, setupNodePoolID, setupNodePoolReplicas)
		if err != nil {
			return microerror.Mask(err)
		}

		err = assertRightNumberOfNodesByVmSize(nodes, setupNodePoolVMSize, setupNodePoolReplicas)
		if err != nil {
			return microerror.Mask(err)
		}

		// Check node pool created in test.
		err = assertRightNumberOfNodesByNodePool(nodes, newNodePoolID, newNodePoolReplicas)
		if err != nil {
			return microerror.Mask(err)
		}

		err = assertRightNumberOfNodesByVmSize(nodes, newNodePoolVMSize, newNodePoolReplicas)
		if err != nil {
			return microerror.Mask(err)
		}

		return nil
	}
	b := backoff.NewConstant(backoff.ShortMaxWait, backoff.ShortMaxInterval)
	err := backoff.Retry(o, b)
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
		return microerror.Mask(unexpectedNumberOfNodesError)
	}

	return nil
}

func assertRightNumberOfNodesByVmSize(nodes *v1.NodeList, vmSize string, expectedNumberOfNodes int) error {
	existingNodesInNodePool, err := getNumberOfNodesByLabel(nodes, LabelVmSize, vmSize)
	if err != nil {
		return microerror.Mask(err)
	}

	if existingNodesInNodePool != expectedNumberOfNodes {
		return microerror.Mask(unexpectedNumberOfNodesError)
	}

	return nil
}

// getNumberOfNodesByLabel returns how many nodes contain the given label with the passed value.
func getNumberOfNodesByLabel(nodes *v1.NodeList, labelName, labelValue string) (int, error) {
	existingNodes := 0

	for _, node := range nodes.Items {
		existingLabelValue, exists := node.GetLabels()[labelName]
		if !exists {
			return 0, microerror.Mask(missingNodePoolLabelError)
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
	nodes, err := k8sclient.CoreV1().Nodes().List(listOptions)
	if err != nil {
		return &v1.NodeList{}, microerror.Mask(err)
	}

	return nodes, nil
}

func (s *Nodepool) CreateNodePool(ctx context.Context, nodepoolID string, replicas int32, vmSize string) error {
	s.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating new node pool %#q with vmsize %#q and %d replicas", nodepoolID, vmSize, replicas))

	var giantSwarmRelease releasev1alpha1.Release
	{
		err := s.ctrlClient.Get(ctx, client.ObjectKey{Namespace: metav1.NamespaceDefault, Name: setup.ReleaseName}, &giantSwarmRelease)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	var azureMachinePool *expcapzv1alpha3.AzureMachinePool
	{
		azureMachinePool = &expcapzv1alpha3.AzureMachinePool{
			TypeMeta: metav1.TypeMeta{
				APIVersion: expcapzv1alpha3.GroupVersion.String(),
				Kind:       "AzureMachinePool",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      nodepoolID,
				Namespace: metav1.NamespaceDefault,
				Labels: map[string]string{
					capiv1alpha3.ClusterLabelName: env.ClusterID(),
					label.AzureOperatorVersion:    env.GetOperatorVersion(),
					label.Cluster:                 env.ClusterID(),
					label.MachinePool:             nodepoolID,
					label.Organization:            "giantswarm",
					label.ReleaseVersion:          strings.TrimPrefix(giantSwarmRelease.GetName(), "v"),
				},
			},
			Spec: expcapzv1alpha3.AzureMachinePoolSpec{
				Location: env.AzureLocation(),
				Template: expcapzv1alpha3.AzureMachineTemplate{
					VMSize:       vmSize,
					SSHPublicKey: base64.StdEncoding.EncodeToString([]byte(env.SSHPublicKey())),
				},
			},
		}

		err := config.K8sClients.CtrlClient().Create(ctx, azureMachinePool)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	{
		var infrastructureCRRef *v1.ObjectReference
		{
			s := runtime.NewScheme()
			err := expcapzv1alpha3.AddToScheme(s)
			if err != nil {
				return microerror.Mask(err)
			}

			infrastructureCRRef, err = reference.GetReference(s, azureMachinePool)
			if err != nil {
				config.Logger.LogCtx(ctx, "level", "warning", fmt.Sprintf("cannot create reference to infrastructure CR: %q", err))
				return microerror.Mask(err)
			}
		}

		clusterOperatorVersion, err := key2.ComponentVersion(giantSwarmRelease, "cluster-operator")
		if err != nil {
			return microerror.Mask(err)
		}

		machinePool := &expcapiv1alpha3.MachinePool{
			TypeMeta: metav1.TypeMeta{
				APIVersion: expcapiv1alpha3.GroupVersion.String(),
				Kind:       "MachinePool",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      nodepoolID,
				Namespace: azureMachinePool.Namespace,
				Labels: map[string]string{
					capiv1alpha3.ClusterLabelName: env.ClusterID(),
					label.AzureOperatorVersion:    env.GetOperatorVersion(),
					label.Cluster:                 env.ClusterID(),
					label.ClusterOperatorVersion:  clusterOperatorVersion,
					label.MachinePool:             nodepoolID,
					label.Organization:            "giantswarm",
					label.ReleaseVersion:          strings.TrimPrefix(giantSwarmRelease.GetName(), "v"),
				},
			},
			Spec: expcapiv1alpha3.MachinePoolSpec{
				ClusterName:    env.ClusterID(),
				Replicas:       to.Int32Ptr(replicas),
				FailureDomains: env.AzureAvailabilityZonesAsStrings(),
				Template: capiv1alpha3.MachineTemplateSpec{
					Spec: capiv1alpha3.MachineSpec{
						ClusterName:       env.ClusterID(),
						InfrastructureRef: *infrastructureCRRef,
					},
				},
			},
		}

		err = config.K8sClients.CtrlClient().Create(ctx, machinePool)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	{
		spark := &v1alpha1.Spark{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "core.giantswarm.io/v1alpha1",
				Kind:       "Spark",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      nodepoolID,
				Namespace: azureMachinePool.Namespace,
				Labels: map[string]string{
					capiv1alpha3.ClusterLabelName: env.ClusterID(),
					label.Cluster:                 env.ClusterID(),
					label.ReleaseVersion:          strings.TrimPrefix(giantSwarmRelease.GetName(), "v"),
				},
			},
		}

		err := config.K8sClients.CtrlClient().Create(ctx, spark)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	s.logger.LogCtx(ctx, "level", "debug", "message", "created new node pool")

	return nil
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
