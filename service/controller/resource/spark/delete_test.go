package spark

import (
	"context"
	"testing"

	corev1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	"github.com/giantswarm/micrologger/microloggertest"
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

	sparkCRShouldNotExistAnymore(ctx, t, ctrlClient)
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

func sparkCRShouldNotExistAnymore(ctx context.Context, t *testing.T, ctrlClient client.Client) {
	var sparkCR corev1alpha1.Spark

	err := ctrlClient.Get(ctx, client.ObjectKey{Namespace: sparkCR.Namespace, Name: sparkCR.Name}, &sparkCR)
	if !errors.IsNotFound(err) {
		t.Fatal("the Spark CR should have been deleted but it's still there")
	}
}
