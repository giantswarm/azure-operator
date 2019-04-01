package blobobject

import (
	"context"
	"reflect"
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2018-07-01/storage"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/certs/certstest"
	"github.com/giantswarm/micrologger/microloggertest"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/giantswarm/azure-operator/service/controller/v7/controllercontext"
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
					K8sClient:             fake.NewSimpleClientset(),
					Logger:                microloggertest.New(),
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

			result, err := newResource.newUpdateChange(ctx, tc.obj, tc.currentState, tc.desiredState)
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
