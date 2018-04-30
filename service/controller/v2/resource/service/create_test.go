package service

import (
	"context"
	"testing"

	corev2 "k8s.io/api/core/v1"
	apismetav2 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/giantswarm/apiextensions/pkg/apis/provider/v2alpha1"
	"github.com/giantswarm/micrologger/microloggertest"
)

func Test_Resource_Service_newCreateChange(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		description     string
		obj             interface{}
		cur             interface{}
		des             interface{}
		expectedService *corev2.Service
	}{
		{
			description: "current state matches desired state, desired state is empty",
			obj: &v2alpha1.AzureConfig{
				Spec: v2alpha1.AzureConfigSpec{
					Cluster: v2alpha1.Cluster{
						ID: "foobar",
					},
				},
			},
			cur: &corev2.Service{
				TypeMeta: apismetav2.TypeMeta{
					Kind:       "Service",
					APIVersion: "v2",
				},
				ObjectMeta: apismetav2.ObjectMeta{
					Name: "master",
				},
			},
			des: &corev2.Service{
				TypeMeta: apismetav2.TypeMeta{
					Kind:       "Service",
					APIVersion: "v2",
				},
				ObjectMeta: apismetav2.ObjectMeta{
					Name: "master",
				},
			},
			expectedService: nil,
		},

		{
			description: "current state is empty, return desired state",
			obj: &v2alpha1.AzureConfig{
				Spec: v2alpha1.AzureConfigSpec{
					Cluster: v2alpha1.Cluster{
						ID: "foobar",
					},
				},
			},
			cur: nil,
			des: &corev2.Service{
				TypeMeta: apismetav2.TypeMeta{
					Kind:       "Service",
					APIVersion: "v2",
				},
				ObjectMeta: apismetav2.ObjectMeta{
					Name: "master",
				},
			},
			expectedService: &corev2.Service{
				TypeMeta: apismetav2.TypeMeta{
					Kind:       "Service",
					APIVersion: "v2",
				},
				ObjectMeta: apismetav2.ObjectMeta{
					Name: "master",
				},
			},
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
			result, err := newResource.newCreateChange(context.TODO(), tc.obj, tc.cur, tc.des)
			if err != nil {
				t.Errorf("expected '%v' got '%#v'", nil, err)
			}
			if tc.expectedService == nil {
				if tc.expectedService != result {
					t.Errorf("expected '%v' got '%v'", tc.expectedService, result)
				}
			} else {
				serviceToCreate, ok := result.(*corev2.Service)
				if !ok {
					t.Errorf("case expected '%T', got '%T'", serviceToCreate, result)
				}
				if tc.expectedService.Name != serviceToCreate.Name {
					t.Errorf("expected %s, got %s", tc.expectedService.Name, serviceToCreate.Name)
				}
			}
		})
	}
}
