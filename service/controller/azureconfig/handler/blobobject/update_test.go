package blobobject

import (
	"context"
	"reflect"
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	providerv1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/provider/v1alpha1"
	g8sfake "github.com/giantswarm/apiextensions/v3/pkg/clientset/versioned/fake"
	"github.com/giantswarm/certs/v3/pkg/certstest"
	"github.com/giantswarm/micrologger/microloggertest"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/giantswarm/azure-operator/v5/service/controller/controllercontext"
	"github.com/giantswarm/azure-operator/v5/service/unittest"
)

const (
	storageAccountNameTest = "storageaccountnametest"
	containerNameTest      = "containernametest"
)

func Test_Resource_ContainerObject_newUpdate(t *testing.T) {
	t.Parallel()
	clusterTpo := providerv1alpha1.AzureConfig{
		Spec: providerv1alpha1.AzureConfigSpec{
			Cluster: providerv1alpha1.Cluster{
				ID: "test-cluster",
			},
		},
	}

	testCases := []struct {
		description   string
		obj           providerv1alpha1.AzureConfig
		currentState  []ContainerObjectState
		desiredState  []ContainerObjectState
		expectedState []ContainerObjectState
	}{
		{
			description:   "current state empty, desired state empty, empty update change",
			obj:           clusterTpo,
			currentState:  []ContainerObjectState{},
			desiredState:  []ContainerObjectState{},
			expectedState: []ContainerObjectState{},
		},
		{
			description:  "current state empty, desired state not empty, not-empty update change",
			obj:          clusterTpo,
			currentState: []ContainerObjectState{},
			desiredState: []ContainerObjectState{
				{
					Body:               "master-body",
					ContainerName:      containerNameTest,
					Key:                prefixMaster,
					StorageAccountName: storageAccountNameTest,
				},
			},
			expectedState: []ContainerObjectState{
				{
					Body:               "master-body",
					ContainerName:      containerNameTest,
					Key:                prefixMaster,
					StorageAccountName: storageAccountNameTest,
				},
			},
		},
		{
			description: "current state matches desired state, empty update change",
			obj:         clusterTpo,
			currentState: []ContainerObjectState{
				{
					Body:               "master-body",
					ContainerName:      containerNameTest,
					Key:                prefixMaster,
					StorageAccountName: storageAccountNameTest,
				},
			},
			desiredState: []ContainerObjectState{
				{
					Body:               "master-body",
					ContainerName:      containerNameTest,
					Key:                prefixMaster,
					StorageAccountName: storageAccountNameTest,
				},
			},
			expectedState: []ContainerObjectState{},
		},
		{
			description: "current state does not match desired state, update container object",
			obj:         clusterTpo,
			currentState: []ContainerObjectState{
				{
					Body:               "master-body",
					ContainerName:      containerNameTest,
					Key:                prefixMaster,
					StorageAccountName: storageAccountNameTest,
				},
			},
			desiredState: []ContainerObjectState{
				{
					Body:               "master-new-body",
					ContainerName:      containerNameTest,
					Key:                prefixMaster,
					StorageAccountName: storageAccountNameTest,
				},
			},
			expectedState: []ContainerObjectState{
				{
					Body:               "master-new-body",
					ContainerName:      containerNameTest,
					Key:                prefixMaster,
					StorageAccountName: storageAccountNameTest,
				},
			},
		},
		{
			description: "current state does not match desired state, update container object",
			obj:         clusterTpo,
			currentState: []ContainerObjectState{
				{
					Body:               "master-body",
					ContainerName:      containerNameTest,
					Key:                prefixMaster,
					StorageAccountName: storageAccountNameTest,
				},
				{
					Body:               "worker-body",
					ContainerName:      containerNameTest,
					Key:                prefixWorker,
					StorageAccountName: storageAccountNameTest,
				},
			},
			desiredState: []ContainerObjectState{
				{
					Body:               "master-new-body",
					ContainerName:      containerNameTest,
					Key:                prefixMaster,
					StorageAccountName: storageAccountNameTest,
				},
				{
					Body:               "worker-body",
					ContainerName:      containerNameTest,
					Key:                prefixWorker,
					StorageAccountName: storageAccountNameTest,
				},
			},
			expectedState: []ContainerObjectState{
				{
					Body:               "master-new-body",
					ContainerName:      containerNameTest,
					Key:                prefixMaster,
					StorageAccountName: storageAccountNameTest,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {

			var err error
			var newResource *Resource
			{
				c := Config{
					CertsSearcher:         certstest.NewSearcher(certstest.Config{}),
					CtrlClient:            unittest.FakeK8sClient().CtrlClient(),
					G8sClient:             g8sfake.NewSimpleClientset(),
					K8sClient:             fake.NewSimpleClientset(),
					Logger:                microloggertest.New(),
					RegistryDomain:        "quay.io",
					StorageAccountsClient: &storage.AccountsClient{},
				}

				newResource, err = New(c)
				if err != nil {
					t.Fatal("expected", nil, "got", err)
				}
			}

			cloudconfig := &CloudConfigMock{}

			ctx := context.TODO()
			c := controllercontext.Context{
				CloudConfig: cloudconfig,
			}
			ctx = controllercontext.NewContext(ctx, c)

			result, err := newResource.newUpdateChange(ctx, tc.currentState, tc.desiredState)
			if err != nil {
				t.Errorf("expected '%v' got '%#v'", nil, err)
			}
			updateChange, ok := result.([]ContainerObjectState)
			if !ok {
				t.Errorf("expected '%T', got '%T'", updateChange, result)
			}

			if !reflect.DeepEqual(tc.expectedState, updateChange) {
				t.Error("expected", tc.expectedState, "got", updateChange)
			}
		})
	}
}
