package endpoints

import (
	"context"
	"testing"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/micrologger/microloggertest"
	corev1 "k8s.io/api/core/v1"
	apismetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func Test_Resource_Endpoints_newDeleteChange(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		description       string
		obj               interface{}
		cur               interface{}
		des               interface{}
		expectedEndpoints *corev1.Endpoints
	}{
		{
			description: "current state matches desired state, return desired state",
			obj: &v1alpha1.AzureConfig{
				Spec: v1alpha1.AzureConfigSpec{
					Cluster: v1alpha1.Cluster{
						ID: "foobar",
					},
				},
			},
			cur: &corev1.Endpoints{
				TypeMeta: apismetav1.TypeMeta{
					Kind:       "Endpoints",
					APIVersion: "v1",
				},
				ObjectMeta: apismetav1.ObjectMeta{
					Name: "master",
				},
			},
			des: &corev1.Endpoints{
				TypeMeta: apismetav1.TypeMeta{
					Kind:       "Endpoints",
					APIVersion: "v1",
				},
				ObjectMeta: apismetav1.ObjectMeta{
					Name: "master",
				},
			},
			expectedEndpoints: &corev1.Endpoints{
				TypeMeta: apismetav1.TypeMeta{
					Kind:       "Endpoints",
					APIVersion: "v1",
				},
				ObjectMeta: apismetav1.ObjectMeta{
					Name: "master",
				},
			},
		},
		{
			description: "current state is empty, no change",
			obj: &v1alpha1.AzureConfig{
				Spec: v1alpha1.AzureConfigSpec{
					Cluster: v1alpha1.Cluster{
						ID: "foobar",
					},
				},
			},
			cur: nil,
			des: &corev1.Endpoints{
				TypeMeta: apismetav1.TypeMeta{
					Kind:       "Service",
					APIVersion: "v1",
				},
				ObjectMeta: apismetav1.ObjectMeta{
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
			K8sClient: fake.NewSimpleClientset(),
			Logger:    microloggertest.New(),
		}
		newResource, err = New(c)
		if err != nil {
			t.Fatal("expected", nil, "got", err)
		}
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			result, err := newResource.newDeleteChange(context.TODO(), tc.cur, tc.des)
			if err != nil {
				t.Errorf("expected '%v' got '%#v'", nil, err)
			}
			if tc.expectedEndpoints == nil {
				if tc.expectedEndpoints != result {
					t.Errorf("expected '%v' got '%v'", tc.expectedEndpoints, result)
				}
			} else {
				endpointsToDelete, ok := result.(*corev1.Endpoints)
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
