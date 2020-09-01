package spark

import (
	"context"
	"testing"

	corev1alpha1 "github.com/giantswarm/apiextensions/v2/pkg/apis/core/v1alpha1"
	"github.com/giantswarm/micrologger/microloggertest"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v4/service/unittest"
)

func TestSparkCRIsDeletedWhenDeletingNodePool(t *testing.T) {
	ctrlClient := unittest.FakeK8sClient().CtrlClient()
	ctx := context.Background()
	logger := microloggertest.New()
	azureMachinePoolNamespace := "default"
	azureMachinePoolName := "my-azure-machine-pool"

	handler := Resource{
		ctrlClient: ctrlClient,
		logger:     logger,
	}

	givenSparkCRInK8sAPI(ctx, t, ctrlClient, azureMachinePoolNamespace, azureMachinePoolName)

	whenNodePoolIsDeleted(ctx, t, handler, azureMachinePoolNamespace, azureMachinePoolName)

	sparkCRShouldNotExistAnymore(ctx, t, ctrlClient, azureMachinePoolNamespace, azureMachinePoolName)
	secretShouldNotExistAnymore(ctx, t, ctrlClient, azureMachinePoolNamespace, secretName(azureMachinePoolName))
}

func TestNoErrorsIfSparkCRDidntExistAlready(t *testing.T) {
	fakeK8sClient := unittest.FakeK8sClient()
	ctx := context.Background()
	logger := microloggertest.New()
	azureMachinePoolNamespace := "default"
	azureMachinePoolName := "my-azure-machine-pool"

	handler := Resource{
		ctrlClient: fakeK8sClient.CtrlClient(),
		logger:     logger,
	}

	whenNodePoolIsDeleted(ctx, t, handler, azureMachinePoolNamespace, azureMachinePoolName)
}

func givenSparkCRInK8sAPI(ctx context.Context, t *testing.T, ctrlClient client.Client, azureMachinePoolNamespace, azureMachinePoolName string) {
	sparkCR := corev1alpha1.Spark{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: azureMachinePoolNamespace,
			Name:      azureMachinePoolName,
		},
		Spec: corev1alpha1.SparkSpec{},
	}
	err := ctrlClient.Create(ctx, &sparkCR)
	if err != nil {
		t.Fatal(err)
	}

	bootstrapSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName(sparkCR.Name),
			Namespace: azureMachinePoolNamespace,
		},
		Data: nil,
		Type: "opaque",
	}
	err = ctrlClient.Create(ctx, &bootstrapSecret)
	if err != nil {
		t.Fatal(err)
	}
}

func whenNodePoolIsDeleted(ctx context.Context, t *testing.T, handler Resource, azureMachinePoolNamespace, azureMachinePoolName string) {
	azureMachinePool := &v1alpha3.AzureMachinePool{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: azureMachinePoolNamespace,
			Name:      azureMachinePoolName,
		},
		Spec: v1alpha3.AzureMachinePoolSpec{},
	}

	err := handler.EnsureDeleted(ctx, azureMachinePool)
	if err != nil {
		t.Fatal(err)
	}
}

func sparkCRShouldNotExistAnymore(ctx context.Context, t *testing.T, ctrlClient client.Client, namespace, name string) {
	var sparkCR corev1alpha1.Spark

	err := ctrlClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, &sparkCR)
	if !errors.IsNotFound(err) {
		t.Fatal("the Spark CR should have been deleted but it's still there")
	}
}

func secretShouldNotExistAnymore(ctx context.Context, t *testing.T, ctrlClient client.Client, namespace, name string) {
	var secret corev1.Secret

	err := ctrlClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, &secret)
	if !errors.IsNotFound(err) {
		t.Fatal("the Bootstrap Secret should have been deleted but it's still there")
	}
}
