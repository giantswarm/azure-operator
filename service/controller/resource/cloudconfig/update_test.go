package cloudconfig

import (
	"context"
	"reflect"
	"testing"
	"time"

	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	g8sfake "github.com/giantswarm/apiextensions/pkg/clientset/versioned/fake"
	"github.com/giantswarm/certs/certstest"
	"github.com/giantswarm/micrologger/microloggertest"
	"github.com/giantswarm/randomkeys"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/giantswarm/azure-operator/v4/client"
	"github.com/giantswarm/azure-operator/v4/pkg/credential"
	"github.com/giantswarm/azure-operator/v4/service/unittest"
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
			logger := microloggertest.New()
			c := client.FactoryConfig{
				CacheDuration:      30 * time.Minute,
				CredentialProvider: credential.EmptyProvider{},
				Logger:             logger,
			}

			clientFactory, err := client.NewFactory(c)
			if err != nil {
				t.Fatal("error creating factory")
			}

			randomkeysconfig := randomkeys.Config{Logger: logger, K8sClient: fake.NewSimpleClientset()}
			searcher, err := randomkeys.NewSearcher(randomkeysconfig)
			if err != nil {
				t.Fatal("error creating randomkeys searcher")
			}

			var newResource *Resource
			{
				c := Config{
					AzureClientsFactory: clientFactory,
					CertsSearcher:       certstest.NewSearcher(certstest.Config{}),
					CredentialProvider:  credential.EmptyProvider{},
					CtrlClient:          unittest.FakeK8sClient().CtrlClient(),
					G8sClient:           g8sfake.NewSimpleClientset(),
					K8sClient:           fake.NewSimpleClientset(),
					RandomKeysSearcher:  searcher,
					Logger:              logger,
					RegistryDomain:      "quay.io",
				}

				newResource, err = New(c)
				if err != nil {
					t.Fatal("expected", nil, "got", err)
				}
			}

			ctx := context.TODO()

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
