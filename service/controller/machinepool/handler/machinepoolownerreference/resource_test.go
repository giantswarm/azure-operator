package machinepoolownerreference

import (
	"context"
	"testing"

	"github.com/giantswarm/micrologger/microloggertest"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	capzexp "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1beta1"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
	capiexp "sigs.k8s.io/cluster-api/exp/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v7/service/unittest"
)

func TestThatMachinePoolAndAzureMachinePoolAreLabeledWithClusterId(t *testing.T) {
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
	thenAzureMachinePoolShouldLabeledWithClusterName(ctx, t, ctrlClient, cluster.Namespace, cluster.Name, azureMachinePool.Name)
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

func whenReconcilingMachinePool(ctx context.Context, ctrlClient client.Client, scheme *runtime.Scheme, machinePool *capiexp.MachinePool) error {
	controller, err := New(Config{
		CtrlClient: ctrlClient,
		Logger:     microloggertest.New(),
		Scheme:     scheme,
	})
	if err != nil {
		return err
	}

	return controller.EnsureCreated(ctx, machinePool)
}

func givenCluster(ctx context.Context, ctrlClient client.Client, clusterNamespace, clusterName string) (*capi.Cluster, error) {
	cluster := &capi.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: clusterNamespace,
			Name:      clusterName,
		},
		Spec: capi.ClusterSpec{},
	}
	err := ctrlClient.Create(ctx, cluster)
	return cluster, err
}

func givenAzureMachinePool(ctx context.Context, ctrlClient client.Client, cluster *capi.Cluster) (*capzexp.AzureMachinePool, error) {
	azureMachinePool := &capzexp.AzureMachinePool{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: cluster.Namespace,
			Name:      cluster.Name,
		},
		Spec: capzexp.AzureMachinePoolSpec{},
	}
	err := ctrlClient.Create(ctx, azureMachinePool)
	return azureMachinePool, err
}

func givenMachinePool(ctx context.Context, ctrlClient client.Client, cluster *capi.Cluster, azureMachinePool *capzexp.AzureMachinePool) (*capiexp.MachinePool, error) {
	machinePool := &capiexp.MachinePool{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: cluster.Namespace,
			Name:      cluster.Name,
		},
		Spec: capiexp.MachinePoolSpec{
			ClusterName: cluster.Name,
			Template: capi.MachineTemplateSpec{
				Spec: capi.MachineSpec{
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
	machinePool := &capiexp.MachinePool{}
	err := ctrlClient.Get(ctx, client.ObjectKey{Namespace: clusterNamespace, Name: machinePoolName}, machinePool)
	if err != nil {
		t.Fatal(err)
	}

	labelClusterName, exists := machinePool.Labels[capi.ClusterLabelName]
	if !exists {
		t.Fatalf("MachinePool should be labeled with Cluster name")
	}

	if labelClusterName != clusterName {
		t.Fatalf("MachinePool is labeled but label contains wrong name")
	}
}

func thenAzureMachinePoolShouldLabeledWithClusterName(ctx context.Context, t *testing.T, ctrlClient client.Client, clusterNamespace, clusterName, azureMachinePoolName string) {
	azureMachinePool := &capzexp.AzureMachinePool{}
	err := ctrlClient.Get(ctx, client.ObjectKey{Namespace: clusterNamespace, Name: azureMachinePoolName}, azureMachinePool)
	if err != nil {
		t.Fatal(err)
	}

	labelClusterName, exists := azureMachinePool.Labels[capi.ClusterLabelName]
	if !exists {
		t.Fatalf("AzureMachinePool should be labeled with Cluster name")
	}

	if labelClusterName != clusterName {
		t.Fatalf("AzureMachinePool is labeled but label contains wrong name")
	}
}

func thenMachinePoolShouldBeOwnedByCluster(ctx context.Context, t *testing.T, ctrlClient client.Client, machinePoolNamespace, machinePoolName string) {
	machinePool := &capiexp.MachinePool{}
	err := ctrlClient.Get(ctx, client.ObjectKey{Namespace: machinePoolNamespace, Name: machinePoolName}, machinePool)
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, ref := range machinePool.OwnerReferences {
		if ref.Kind == "Cluster" && ref.APIVersion == capi.GroupVersion.String() {
			found = true
		}
	}

	if !found {
		t.Fatalf("MachinePool should be owned by Cluster in OwnerReferences")
	}
}

func thenAzureMachinePoolShouldBeOwnedByMachinePool(ctx context.Context, t *testing.T, ctrlClient client.Client, azureMachinePoolNamespace, azureMachinePoolName string) {
	azureMachinePool := &capzexp.AzureMachinePool{}
	err := ctrlClient.Get(ctx, client.ObjectKey{Namespace: azureMachinePoolNamespace, Name: azureMachinePoolName}, azureMachinePool)
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, ref := range azureMachinePool.OwnerReferences {
		if ref.Kind == "MachinePool" && ref.APIVersion == capiexp.GroupVersion.String() && *ref.Controller {
			found = true
		}
	}

	if !found {
		t.Fatalf("AzureMachinePool should be owned by MachinePool in OwnerReferences")
	}
}
