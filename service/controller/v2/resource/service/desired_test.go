package service

import (
	"context"
	"testing"

	corev2 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/giantswarm/apiextensions/pkg/apis/provider/v2alpha1"
	"github.com/giantswarm/micrologger/microloggertest"
)

func Test_Resource_Service_GetDesiredState(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		description        string
		obj                interface{}
		expectedNamespace  string
		expectedName       string
		expectedPort       int
		expectedTargetPort string
	}{
		{
			description: "Get service from custom object",
			obj: &v2alpha1.AzureConfig{
				Spec: v2alpha1.AzureConfigSpec{
					Cluster: v2alpha1.Cluster{
						ID: "al9qy",
					},
				},
			},
			expectedNamespace:  "al9qy",
			expectedName:       "master",
			expectedPort:       443,
			expectedTargetPort: "443",
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
			result, err := newResource.GetDesiredState(context.TODO(), tc.obj)
			if err != nil {
				t.Errorf("expected '%v' got '%#v'", nil, err)
			}
			desiredService, ok := result.(*corev2.Service)
			if !ok {
				t.Errorf("case expected '%T', got '%T'", desiredService, result)
			}

			if tc.expectedNamespace != desiredService.ObjectMeta.Namespace {
				t.Errorf("expected namespace %q got %q", tc.expectedNamespace, desiredService.ObjectMeta.Namespace)
			}

			if tc.expectedName != desiredService.ObjectMeta.Name {
				t.Errorf("expected name %q got %q", tc.expectedName, desiredService.ObjectMeta.Name)
			}

			if int32(tc.expectedPort) != desiredService.Spec.Ports[0].Port {
				t.Errorf("expected port %q got %q", int32(tc.expectedPort), desiredService.Spec.Ports[0].Port)
			}

			if intstr.FromInt(tc.expectedPort) != desiredService.Spec.Ports[0].TargetPort {
				t.Errorf("expected target port %q got %q", intstr.FromInt(tc.expectedPort), desiredService.Spec.Ports[0].TargetPort)
			}
		})
	}
}
