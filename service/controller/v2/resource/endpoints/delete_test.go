package endpoints

import (
	"context"
	"testing"

	corev2 "k8s.io/api/core/v1"
	apismetav2 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/giantswarm/apiextensions/pkg/apis/provider/v2alpha1"
	"github.com/giantswarm/micrologger/microloggertest"

	"github.com/giantswarm/azure-operator/client/fakeclient"
)

func Test_Resource_Endpoints_newDeleteChange(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		description       string
		obj               interface{}
		cur               interface{}
		des               interface{}
		expectedEndpoints *corev2.Endpoints
	}{
		{
			description: "current state matches desired state, return desired state",
			obj: &v2alpha1.AzureConfig{
				Spec: v2alpha1.AzureConfigSpec{
					Cluster: v2alpha1.Cluster{
						ID: "foobar",
					},
				},
			},
			cur: &corev2.Endpoints{
				TypeMeta: apismetav2.TypeMeta{
					Kind:       "Endpoints",
					APIVersion: "v2",
				},
				ObjectMeta: apismetav2.ObjectMeta{
					Name: "master",
				},
			},
			des: &corev2.Endpoints{
				TypeMeta: apismetav2.TypeMeta{
					Kind:       "Endpoints",
					APIVersion: "v2",
				},
				ObjectMeta: apismetav2.ObjectMeta{
					Name: "master",
				},
			},
			expectedEndpoints: &corev2.Endpoints{
				TypeMeta: apismetav2.TypeMeta{
					Kind:       "Endpoints",
					APIVersion: "v2",
				},
				ObjectMeta: apismetav2.ObjectMeta{
					Name: "master",
				},
			},
		},
		{
			description: "current state is empty, no change",
			obj: &v2alpha1.AzureConfig{
				Spec: v2alpha1.AzureConfigSpec{
					Cluster: v2alpha1.Cluster{
						ID: "foobar",
					},
				},
			},
			cur: nil,
			des: &corev2.Endpoints{
				TypeMeta: apismetav2.TypeMeta{
					Kind:       "Service",
					APIVersion: "v2",
				},
				ObjectMeta: apismetav2.ObjectMeta{
					Name: "master",
				},
			},
			expectedEndpoints: nil,
		},
	}

	var err error
	var newResource *Resource
	{
		c := Config{
			AzureConfig: fakeclient.NewAzureConfig(),
			K8sClient:   fake.NewSimpleClientset(),
			Logger:      microloggertest.New(),
		}
		newResource, err = New(c)
		if err != nil {
			t.Fatal("expected", nil, "got", err)
		}
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			result, err := newResource.newDeleteChange(context.TODO(), tc.obj, tc.cur, tc.des)
			if err != nil {
				t.Errorf("expected '%v' got '%#v'", nil, err)
			}
			if tc.expectedEndpoints == nil {
				if tc.expectedEndpoints != result {
					t.Errorf("expected '%v' got '%v'", tc.expectedEndpoints, result)
				}
			} else {
				endpointsToDelete, ok := result.(*corev2.Endpoints)
				if !ok {
					t.Errorf("case expected '%T', got '%T'", endpointsToDelete, result)
				}
				if tc.expectedEndpoints.Name != endpointsToDelete.Name {
					t.Errorf("expected %s, got %s", tc.expectedEndpoints.Name, endpointsToDelete.Name)
				}
			}
		})
	}
}
