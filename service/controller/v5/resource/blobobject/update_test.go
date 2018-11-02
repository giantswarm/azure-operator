package blobobject

import (
	"context"
	"reflect"
	"testing"

	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/controller/v5/controllercontext"
	"github.com/giantswarm/micrologger/microloggertest"
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
		currentState  map[string]ContainerObjectState
		desiredState  map[string]ContainerObjectState
		expectedState map[string]ContainerObjectState
	}{
		{
			description:   "current state empty, desired state empty, empty update change",
			obj:           clusterTpo,
			currentState:  map[string]ContainerObjectState{},
			desiredState:  map[string]ContainerObjectState{},
			expectedState: map[string]ContainerObjectState{},
		},
		{
			description:  "current state empty, desired state not empty, not-empty update change",
			obj:          clusterTpo,
			currentState: map[string]ContainerObjectState{},
			desiredState: map[string]ContainerObjectState{
				"master": {
					Body:               "master-body",
					ContainerName:      containerNameTest,
					Key:                prefixMaster,
					StorageAccountName: storageAccountNameTest,
				},
			},
			expectedState: map[string]ContainerObjectState{
				"master": {},
			},
		},
		{
			description: "current state matches desired state, empty update change",
			obj:         clusterTpo,
			currentState: map[string]ContainerObjectState{
				"master": {
					Body:               "master-body",
					ContainerName:      containerNameTest,
					Key:                prefixMaster,
					StorageAccountName: storageAccountNameTest,
				},
			},
			desiredState: map[string]ContainerObjectState{
				"master": {
					Body:               "master-body",
					ContainerName:      containerNameTest,
					Key:                prefixMaster,
					StorageAccountName: storageAccountNameTest,
				},
			},
			expectedState: map[string]ContainerObjectState{
				"master": {},
			},
		},
		{
			description: "current state does not match desired state, update container object",
			obj:         clusterTpo,
			currentState: map[string]ContainerObjectState{
				"master": {
					Body:               "master-body",
					ContainerName:      containerNameTest,
					Key:                prefixMaster,
					StorageAccountName: storageAccountNameTest,
				},
			},
			desiredState: map[string]ContainerObjectState{
				"master": {
					Body:               "master-new-body",
					ContainerName:      containerNameTest,
					Key:                prefixMaster,
					StorageAccountName: storageAccountNameTest,
				},
			},
			expectedState: map[string]ContainerObjectState{
				"master": {
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
			currentState: map[string]ContainerObjectState{
				"master": {
					Body:               "master-body",
					ContainerName:      containerNameTest,
					Key:                prefixMaster,
					StorageAccountName: storageAccountNameTest,
				},
				"worker": {
					Body:               "worker-body",
					ContainerName:      containerNameTest,
					Key:                prefixWorker,
					StorageAccountName: storageAccountNameTest,
				},
			},
			desiredState: map[string]ContainerObjectState{
				"master": {
					Body:               "master-new-body",
					ContainerName:      containerNameTest,
					Key:                prefixMaster,
					StorageAccountName: storageAccountNameTest,
				},
				"worker": {
					Body:               "worker-body",
					ContainerName:      containerNameTest,
					Key:                prefixWorker,
					StorageAccountName: storageAccountNameTest,
				},
			},
			expectedState: map[string]ContainerObjectState{
				"master": {
					Body:               "master-new-body",
					ContainerName:      containerNameTest,
					Key:                prefixMaster,
					StorageAccountName: storageAccountNameTest,
				},
				"worker": {},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {

			var err error
			var newResource *Resource
			{
				c := Config{}
				c.Logger = microloggertest.New()
				c.HostAzureClientSetConfig = client.AzureClientSetConfig{
					ClientID:       "clientid",
					ClientSecret:   "clientsecret",
					SubscriptionID: "subscriptionid",
					TenantID:       "tenantid",
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
			updateChange, ok := result.(map[string]ContainerObjectState)
			if !ok {
				t.Errorf("expected '%T', got '%T'", updateChange, result)
			}

			if !reflect.DeepEqual(tc.expectedState, updateChange) {
				t.Error("expected", tc.expectedState, "got", updateChange)
			}
		})
	}
}
