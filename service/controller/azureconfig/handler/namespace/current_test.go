package namespace

import (
	"context"
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/giantswarm/apiextensions/v5/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/micrologger/microloggertest"
)

func Test_Resource_Namespace_GetCurrentState(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		Obj               interface{}
		ExpectedNamespace *corev1.Namespace
	}{
		{
			Obj: &v1alpha1.AzureConfig{
				Spec: v1alpha1.AzureConfigSpec{
					Cluster: v1alpha1.Cluster{
						ID: "al9qy",
					},
				},
			},
			ExpectedNamespace: nil,
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

	for i, tc := range testCases {
		result, err := newResource.GetCurrentState(context.TODO(), tc.Obj)
		if err != nil {
			t.Fatal("case", i+1, "expected", nil, "got", err)
		}
		if !reflect.DeepEqual(tc.ExpectedNamespace, result) {
			t.Fatalf("case %d expected %#v got %#v", i+1, tc.ExpectedNamespace, result)
		}
	}
}
