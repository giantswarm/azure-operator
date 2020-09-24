package nodestatus

import (
	"context"
	"fmt"
	"testing"

	"github.com/giantswarm/apiextensions/v2/pkg/label"
	"github.com/giantswarm/micrologger/microloggertest"
	"github.com/golang/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	expcapzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	expcapiv1alpha3 "sigs.k8s.io/cluster-api/exp/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v4/pkg/mock/mock_tenantcluster"
	"github.com/giantswarm/azure-operator/v4/service/unittest"
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
		givenReadyNode("worker1"),
		givenReadyNode("worker2"),
		givenReadyNode("worker3"),
		givenReadyNode("worker4"),
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
		givenReadyNode("worker1"),
		givenReadyNode("worker2"),
	}
	notReadyNodes := []corev1.Node{
		givenNotReadyNode("worker3"),
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

func givenNodePool(ctx context.Context, ctrclient client.Client, ready bool, replicas int) (*expcapiv1alpha3.MachinePool, *expcapzv1alpha3.AzureMachinePool, error) {
	cluster := &capiv1alpha3.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "clustername",
			Namespace: metav1.NamespaceDefault,
			Labels: map[string]string{
				label.Cluster:                 "clustername",
				capiv1alpha3.ClusterLabelName: "clustername",
			},
		},
		Spec: capiv1alpha3.ClusterSpec{},
	}
	err := ctrclient.Create(ctx, cluster)
	if err != nil {
		return &expcapiv1alpha3.MachinePool{}, &expcapzv1alpha3.AzureMachinePool{}, err
	}

	azureMachinePool := &expcapzv1alpha3.AzureMachinePool{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: metav1.NamespaceDefault,
			Name:      "my-azure-machine-pool",
			Labels: map[string]string{
				label.Cluster:                 "clustername",
				capiv1alpha3.ClusterLabelName: "clustername",
			},
		},
		Spec: expcapzv1alpha3.AzureMachinePoolSpec{
			ProviderIDList: []string{"azure://worker1", "azure://worker2", "azure://worker3", "azure://worker4"},
		},
		Status: expcapzv1alpha3.AzureMachinePoolStatus{
			Ready:             ready,
			Replicas:          int32(replicas),
			ProvisioningState: nil,
			FailureReason:     nil,
			FailureMessage:    nil,
		},
	}

	err = ctrclient.Create(ctx, azureMachinePool)
	if err != nil {
		return &expcapiv1alpha3.MachinePool{}, &expcapzv1alpha3.AzureMachinePool{}, err
	}

	machinePool := &expcapiv1alpha3.MachinePool{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: metav1.NamespaceDefault,
			Name:      "my-machine-pool",
			Labels: map[string]string{
				label.Cluster:                 "clustername",
				capiv1alpha3.ClusterLabelName: "clustername",
			},
		},
		Spec: expcapiv1alpha3.MachinePoolSpec{
			Template: capiv1alpha3.MachineTemplateSpec{
				Spec: capiv1alpha3.MachineSpec{
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
		return &expcapiv1alpha3.MachinePool{}, &expcapzv1alpha3.AzureMachinePool{}, err
	}

	return machinePool, azureMachinePool, nil
}

func givenNodes(ctx context.Context, ctrlClient client.Client, nodes []corev1.Node) error {
	for _, node := range nodes {
		err := ctrlClient.Create(ctx, &node)
		if err != nil {
			return err
		}
	}

	return nil
}

func givenReadyNode(name string) corev1.Node {
	return corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
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

func givenNotReadyNode(name string) corev1.Node {
	return corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: corev1.NodeSpec{
			ProviderID:    fmt.Sprintf("azure://%s", name),
			Unschedulable: false,
		},
		Status: corev1.NodeStatus{},
	}
}
