package controller

import (
	"context"
	"testing"

	"github.com/giantswarm/micrologger/microloggertest"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v4/service/unittest"
)

func TestThatAzureClusterIsLabeledWithClusterId(t *testing.T) {
	ctx := context.Background()
	fakeK8sClient := unittest.FakeK8sClient()
	ctrlClient := fakeK8sClient.CtrlClient()
	controller, err := NewClusterOwnerReferences(ClusterOwnerReferencesConfig{
		CtrlClient: ctrlClient,
		Logger:     microloggertest.New(),
		Scheme:     fakeK8sClient.Scheme(),
	})
	if err != nil {
		t.Fatal(err)
	}

	clusterNamespace := "default"
	clusterName := "my-cluster"
	cluster := &capiv1alpha3.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: clusterNamespace,
			Name:      clusterName,
		},
		Spec: capiv1alpha3.ClusterSpec{
			InfrastructureRef: &v1.ObjectReference{
				Kind:      "AzureCluster",
				Namespace: clusterNamespace,
				Name:      clusterName,
			},
		},
	}
	err = ctrlClient.Create(ctx, cluster)
	if err != nil {
		t.Fatal(err)
	}

	azureCluster := &v1alpha3.AzureCluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: clusterNamespace,
			Name:      clusterName,
		},
		Spec: v1alpha3.AzureClusterSpec{},
	}
	err = ctrlClient.Create(ctx, azureCluster)
	if err != nil {
		t.Fatal(err)
	}

	err = controller.EnsureCreated(ctx, cluster)
	if err != nil {
		t.Fatal(err)
	}

	err = ctrlClient.Get(ctx, client.ObjectKey{Namespace: clusterNamespace, Name: clusterName}, azureCluster)
	if err != nil {
		t.Fatal(err)
	}

	labelClusterName, exists := azureCluster.Labels[capiv1alpha3.ClusterLabelName]
	if !exists {
		t.Fatalf("Azure cluster should be labeled with Cluster name")
	}

	if labelClusterName != clusterName {
		t.Fatalf("Azure cluster is labeled but label contains wrong name")
	}
}

func TestThatAzureClusterIsOwnedByCluster(t *testing.T) {
	ctx := context.Background()
	fakeK8sClient := unittest.FakeK8sClient()
	ctrlClient := fakeK8sClient.CtrlClient()
	scheme := fakeK8sClient.Scheme()

	azureCluster, err := givenAzureCluster(ctx, ctrlClient, "default", "my-cluster")
	if err != nil {
		t.Fatal(err)
	}

	cluster := &capiv1alpha3.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "my-cluster",
		},
		Spec: capiv1alpha3.ClusterSpec{
			InfrastructureRef: &v1.ObjectReference{
				Kind:      "AzureCluster",
				Namespace: azureCluster.Namespace,
				Name:      azureCluster.Name,
			},
		},
	}
	err = ctrlClient.Create(ctx, cluster)
	if err != nil {
		t.Fatal(err)
	}

	err = whenReconcilingCluster(ctx, ctrlClient, scheme, cluster)
	if err != nil {
		t.Fatal(err)
	}

	thenAzureClusterShouldBeOwnedByCluster(ctx, t, ctrlClient, azureCluster.Namespace, azureCluster.Name)
}

func givenAzureCluster(ctx context.Context, ctrlClient client.Client, clusterNamespace, clusterName string) (*v1alpha3.AzureCluster, error) {
	azureCluster := &v1alpha3.AzureCluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: clusterNamespace,
			Name:      clusterName,
		},
		Spec: v1alpha3.AzureClusterSpec{},
	}
	err := ctrlClient.Create(ctx, azureCluster)
	return azureCluster, err
}

func whenReconcilingCluster(ctx context.Context, ctrlClient client.Client, scheme *runtime.Scheme, cluster *capiv1alpha3.Cluster) error {
	controller, err := NewClusterOwnerReferences(ClusterOwnerReferencesConfig{
		CtrlClient: ctrlClient,
		Logger:     microloggertest.New(),
		Scheme:     scheme,
	})
	if err != nil {
		return err
	}

	return controller.EnsureCreated(ctx, cluster)
}

func thenAzureClusterShouldBeOwnedByCluster(ctx context.Context, t *testing.T, ctrlClient client.Client, azureClusterNamespace, azureClusterName string) {
	azureCluster := &v1alpha3.AzureCluster{}
	err := ctrlClient.Get(ctx, client.ObjectKey{Namespace: azureClusterNamespace, Name: azureClusterName}, azureCluster)
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, ref := range azureCluster.OwnerReferences {
		if ref.Kind == "Cluster" && ref.APIVersion == capiv1alpha3.GroupVersion.String() {
			found = true
		}
	}

	if !found {
		t.Fatalf("Azure cluster should be owned by Cluster in OwnerReferences")
	}
}
