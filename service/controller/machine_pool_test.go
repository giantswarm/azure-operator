package controller

import (
	"context"
	"testing"

	"github.com/giantswarm/micrologger/microloggertest"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	expcapzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	expcapiv1alpha3 "sigs.k8s.io/cluster-api/exp/api/v1alpha3"
	v1alpha32 "sigs.k8s.io/cluster-api/exp/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v4/service/unittest"
)

func TestThatMachinePoolIsLabeledWithClusterId(t *testing.T) {
	ctx := context.Background()
	fakeK8sClient := unittest.FakeK8sClient()
	ctrlClient := fakeK8sClient.CtrlClient()
	scheme := fakeK8sClient.Scheme()

	cluster, err := givenCluster(ctx, ctrlClient, "default", "my-cluster")
	if err != nil {
		t.Fatal(err)
	}

	azureMachinePool, err := givenAzureMachinePool(ctx, ctrlClient, cluster)
	if err != nil {
		t.Fatal(err)
	}

	machinePool, err := givenMachinePool(ctx, ctrlClient, cluster, azureMachinePool)
	if err != nil {
		t.Fatal(err)
	}

	err = whenReconcilingMachinePool(ctx, ctrlClient, scheme, machinePool)
	if err != nil {
		t.Fatal(err)
	}

	thenMachinePoolShouldLabeledWithClusterName(ctx, t, ctrlClient, cluster.Namespace, cluster.Name, machinePool.Name)
}

func TestThatMachinePoolIsOwnedByCluster(t *testing.T) {
	ctx := context.Background()
	fakeK8sClient := unittest.FakeK8sClient()
	ctrlClient := fakeK8sClient.CtrlClient()
	scheme := fakeK8sClient.Scheme()

	cluster, err := givenCluster(ctx, ctrlClient, "default", "my-cluster")
	if err != nil {
		t.Fatal(err)
	}

	azureMachinePool, err := givenAzureMachinePool(ctx, ctrlClient, cluster)
	if err != nil {
		t.Fatal(err)
	}

	machinePool, err := givenMachinePool(ctx, ctrlClient, cluster, azureMachinePool)
	if err != nil {
		t.Fatal(err)
	}

	err = whenReconcilingMachinePool(ctx, ctrlClient, scheme, machinePool)
	if err != nil {
		t.Fatal(err)
	}

	thenMachinePoolShouldBeOwnedByCluster(ctx, t, ctrlClient, cluster.Namespace, machinePool.Name)
	thenAzureMachinePoolShouldBeOwnedByMachinePool(ctx, t, ctrlClient, cluster.Namespace, azureMachinePool.Name)
}

func whenReconcilingMachinePool(ctx context.Context, ctrlClient client.Client, scheme *runtime.Scheme, machinePool *v1alpha32.MachinePool) error {
	controller, err := NewMachinePoolOwnerReferences(MachinePoolOwnerReferencesConfig{
		CtrlClient: ctrlClient,
		Logger:     microloggertest.New(),
		Scheme:     scheme,
	})
	if err != nil {
		return err
	}

	return controller.EnsureCreated(ctx, machinePool)
}

func givenCluster(ctx context.Context, ctrlClient client.Client, clusterNamespace, clusterName string) (*capiv1alpha3.Cluster, error) {
	cluster := &capiv1alpha3.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: clusterNamespace,
			Name:      clusterName,
		},
		Spec: capiv1alpha3.ClusterSpec{},
	}
	err := ctrlClient.Create(ctx, cluster)
	return cluster, err
}

func givenAzureMachinePool(ctx context.Context, ctrlClient client.Client, cluster *capiv1alpha3.Cluster) (*expcapzv1alpha3.AzureMachinePool, error) {
	azureMachinePool := &expcapzv1alpha3.AzureMachinePool{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: cluster.Namespace,
			Name:      cluster.Name,
		},
		Spec: expcapzv1alpha3.AzureMachinePoolSpec{},
	}
	err := ctrlClient.Create(ctx, azureMachinePool)
	return azureMachinePool, err
}

func givenMachinePool(ctx context.Context, ctrlClient client.Client, cluster *capiv1alpha3.Cluster, azureMachinePool *expcapzv1alpha3.AzureMachinePool) (*v1alpha32.MachinePool, error) {
	machinePool := &v1alpha32.MachinePool{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: cluster.Namespace,
			Name:      cluster.Name,
		},
		Spec: v1alpha32.MachinePoolSpec{
			ClusterName: cluster.Name,
			Template: capiv1alpha3.MachineTemplateSpec{
				Spec: capiv1alpha3.MachineSpec{
					InfrastructureRef: v1.ObjectReference{
						Kind:      "AzureMachinePool",
						Namespace: azureMachinePool.Namespace,
						Name:      azureMachinePool.Name,
					},
				},
			},
		},
	}
	err := ctrlClient.Create(ctx, machinePool)
	return machinePool, err
}

func thenMachinePoolShouldLabeledWithClusterName(ctx context.Context, t *testing.T, ctrlClient client.Client, clusterNamespace, clusterName, machinePoolName string) {
	machinePool := &v1alpha32.MachinePool{}
	err := ctrlClient.Get(ctx, client.ObjectKey{Namespace: clusterNamespace, Name: machinePoolName}, machinePool)
	if err != nil {
		t.Fatal(err)
	}

	labelClusterName, exists := machinePool.Labels[capiv1alpha3.ClusterLabelName]
	if !exists {
		t.Fatalf("MachinePool should be labeled with Cluster name")
	}

	if labelClusterName != clusterName {
		t.Fatalf("MachinePool is labeled but label contains wrong name")
	}
}

func thenMachinePoolShouldBeOwnedByCluster(ctx context.Context, t *testing.T, ctrlClient client.Client, machinePoolNamespace, machinePoolName string) {
	machinePool := &v1alpha32.MachinePool{}
	err := ctrlClient.Get(ctx, client.ObjectKey{Namespace: machinePoolNamespace, Name: machinePoolName}, machinePool)
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, ref := range machinePool.OwnerReferences {
		if ref.Kind == "Cluster" && ref.APIVersion == capiv1alpha3.GroupVersion.String() {
			found = true
		}
	}

	if !found {
		t.Fatalf("MachinePool should be owned by Cluster in OwnerReferences")
	}
}

func thenAzureMachinePoolShouldBeOwnedByMachinePool(ctx context.Context, t *testing.T, ctrlClient client.Client, azureMachinePoolNamespace, azureMachinePoolName string) {
	azureMachinePool := &expcapzv1alpha3.AzureMachinePool{}
	err := ctrlClient.Get(ctx, client.ObjectKey{Namespace: azureMachinePoolNamespace, Name: azureMachinePoolName}, azureMachinePool)
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, ref := range azureMachinePool.OwnerReferences {
		if ref.Kind == "MachinePool" && ref.APIVersion == expcapiv1alpha3.GroupVersion.String() && *ref.Controller {
			found = true
		}
	}

	if !found {
		t.Fatalf("AzureMachinePool should be owned by MachinePool in OwnerReferences")
	}
}
