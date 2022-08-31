package nodestatus

import (
	"context"
	"fmt"
	"testing"

	"github.com/giantswarm/apiextensions/v6/pkg/label"
	"github.com/giantswarm/micrologger/microloggertest"
	"github.com/golang/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capzexp "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1beta1"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
	capiexp "sigs.k8s.io/cluster-api/exp/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v6/pkg/mock/mock_tenantcluster"
	"github.com/giantswarm/azure-operator/v6/service/unittest"
)

func Test_NodeStatusIsSaved(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fakeClient := unittest.FakeK8sClient()
	ctrlClient := fakeClient.CtrlClient()

	fakeTenantClient := unittest.FakeK8sClient()
	ctrlClientTenant := fakeTenantClient.CtrlClient()

	mockTenantClientFactory := mock_tenantcluster.NewMockFactory(ctrl)
	mockTenantClientFactory.
		EXPECT().
		GetClient(gomock.Any(), gomock.Any()).
		Return(ctrlClientTenant, nil).
		Times(1)

	config := Config{
		CtrlClient:          ctrlClient,
		Logger:              microloggertest.New(),
		TenantClientFactory: mockTenantClientFactory,
	}
	handler, err := New(config)
	if err != nil {
		t.Fatal(err)
	}

	nodes := []corev1.Node{
		givenReadyNode("worker1", "my-machine-pool"),
		givenReadyNode("worker2", "my-machine-pool"),
		givenReadyNode("worker3", "my-machine-pool"),
		givenReadyNode("worker4", "my-machine-pool"),
	}
	err = givenNodes(ctx, ctrlClientTenant, nodes)
	if err != nil {
		t.Fatal(err)
	}

	expectedAvailableReplicas := len(nodes)
	expectedUnavailableReplicas := 0
	machinePool, _, err := givenNodePool(ctx, ctrlClient, true, expectedAvailableReplicas)
	if err != nil {
		t.Fatal(err)
	}

	err = handler.EnsureCreated(ctx, machinePool)
	if err != nil {
		t.Fatal(err)
	}

	err = ctrlClient.Get(ctx, client.ObjectKey{Namespace: machinePool.Namespace, Name: machinePool.Name}, machinePool)
	if err != nil {
		t.Fatal(err)
	}

	if machinePool.Status.ReadyReplicas != int32(len(nodes)) {
		t.Fatalf("Wrong number of ready replicas in MachinePool status field. Expected %d, got %d", expectedAvailableReplicas, machinePool.Status.ReadyReplicas)
	}

	if machinePool.Status.UnavailableReplicas != 0 {
		t.Fatalf("Wrong number of unavailable replicas in MachinePool status field. Expected %d, got %d", expectedUnavailableReplicas, machinePool.Status.UnavailableReplicas)
	}
}

func Test_NodeStatusIsSavedWhenThereIsOneNodeNotReady(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fakeClient := unittest.FakeK8sClient()
	ctrlClient := fakeClient.CtrlClient()

	fakeTenantClient := unittest.FakeK8sClient()
	ctrlClientTenant := fakeTenantClient.CtrlClient()

	mockTenantClientFactory := mock_tenantcluster.NewMockFactory(ctrl)
	mockTenantClientFactory.
		EXPECT().
		GetClient(gomock.Any(), gomock.Any()).
		Return(ctrlClientTenant, nil).
		Times(1)

	config := Config{
		CtrlClient:          ctrlClient,
		Logger:              microloggertest.New(),
		TenantClientFactory: mockTenantClientFactory,
	}
	handler, err := New(config)
	if err != nil {
		t.Fatal(err)
	}

	readyNodes := []corev1.Node{
		givenReadyNode("worker1", "my-machine-pool"),
		givenReadyNode("worker2", "my-machine-pool"),
	}
	notReadyNodes := []corev1.Node{
		givenNotReadyNode("worker3", "my-machine-pool"),
	}
	var nodes []corev1.Node
	nodes = append(nodes, readyNodes...)
	nodes = append(nodes, notReadyNodes...)
	err = givenNodes(ctx, ctrlClientTenant, nodes)
	if err != nil {
		t.Fatal(err)
	}

	expectedAvailableReplicas := len(nodes)
	expectedUnavailableReplicas := 1
	machinePool, _, err := givenNodePool(ctx, ctrlClient, true, expectedAvailableReplicas+expectedUnavailableReplicas)
	if err != nil {
		t.Fatal(err)
	}

	err = handler.EnsureCreated(ctx, machinePool)
	if err != nil {
		t.Fatal(err)
	}

	err = ctrlClient.Get(ctx, client.ObjectKey{Namespace: machinePool.Namespace, Name: machinePool.Name}, machinePool)
	if err != nil {
		t.Fatal(err)
	}

	if machinePool.Status.ReadyReplicas != int32(len(readyNodes)) {
		t.Fatalf("Wrong number of ready replicas in MachinePool status field. Expected %d, got %d", len(readyNodes), machinePool.Status.ReadyReplicas)
	}

	if machinePool.Status.AvailableReplicas != int32(len(readyNodes)+len(notReadyNodes)) {
		t.Fatalf("Wrong number of available replicas in MachinePool status field. Expected %d, got %d", expectedAvailableReplicas, machinePool.Status.AvailableReplicas)
	}

	if machinePool.Status.UnavailableReplicas != 1 {
		t.Fatalf("Wrong number of unavailable replicas in MachinePool status field. Expected %d, got %d", expectedUnavailableReplicas, machinePool.Status.UnavailableReplicas)
	}
}

func givenNodePool(ctx context.Context, ctrclient client.Client, ready bool, replicas int) (*capiexp.MachinePool, *capzexp.AzureMachinePool, error) {
	cluster := &capi.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "clustername",
			Namespace: metav1.NamespaceDefault,
			Labels: map[string]string{
				label.Cluster:         "clustername",
				capi.ClusterLabelName: "clustername",
			},
		},
		Spec: capi.ClusterSpec{},
	}
	err := ctrclient.Create(ctx, cluster)
	if err != nil {
		return &capiexp.MachinePool{}, &capzexp.AzureMachinePool{}, err
	}

	azureMachinePool := &capzexp.AzureMachinePool{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: metav1.NamespaceDefault,
			Name:      "my-azure-machine-pool",
			Labels: map[string]string{
				label.Cluster:         "clustername",
				capi.ClusterLabelName: "clustername",
			},
		},
		Spec: capzexp.AzureMachinePoolSpec{
			ProviderIDList: []string{"azure://worker1", "azure://worker2", "azure://worker3", "azure://worker4"},
		},
		Status: capzexp.AzureMachinePoolStatus{
			Ready:             ready,
			Replicas:          int32(replicas),
			ProvisioningState: nil,
			FailureReason:     nil,
			FailureMessage:    nil,
		},
	}

	err = ctrclient.Create(ctx, azureMachinePool)
	if err != nil {
		return &capiexp.MachinePool{}, &capzexp.AzureMachinePool{}, err
	}

	machinePool := &capiexp.MachinePool{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: metav1.NamespaceDefault,
			Name:      "my-machine-pool",
			Labels: map[string]string{
				label.Cluster:         "clustername",
				capi.ClusterLabelName: "clustername",
			},
		},
		Spec: capiexp.MachinePoolSpec{
			Template: capi.MachineTemplateSpec{
				Spec: capi.MachineSpec{
					InfrastructureRef: corev1.ObjectReference{
						Kind:      azureMachinePool.Kind,
						Namespace: metav1.NamespaceDefault,
						Name:      azureMachinePool.Name,
					},
				},
			},
		},
	}

	err = ctrclient.Create(ctx, machinePool)
	if err != nil {
		return &capiexp.MachinePool{}, &capzexp.AzureMachinePool{}, err
	}

	return machinePool, azureMachinePool, nil
}

func givenNodes(ctx context.Context, ctrlClient client.Client, nodes []corev1.Node) error {
	for i := range nodes {
		err := ctrlClient.Create(ctx, &nodes[i])
		if err != nil {
			return err
		}
	}

	return nil
}

func givenReadyNode(name, machinePoolName string) corev1.Node {
	return corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				label.MachinePool: machinePoolName,
			},
		},
		Spec: corev1.NodeSpec{
			ProviderID:    fmt.Sprintf("azure://%s", name),
			Unschedulable: false,
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Type:   corev1.NodeReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}
}

func givenNotReadyNode(name, machinePoolName string) corev1.Node {
	return corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				label.MachinePool: machinePoolName,
			},
		},
		Spec: corev1.NodeSpec{
			ProviderID:    fmt.Sprintf("azure://%s", name),
			Unschedulable: false,
		},
		Status: corev1.NodeStatus{},
	}
}
