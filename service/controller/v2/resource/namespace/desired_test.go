package namespace

import (
	"context"
	"testing"

	corev2 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/giantswarm/apiextensions/pkg/apis/provider/v2alpha1"
	"github.com/giantswarm/micrologger/microloggertest"
)

func Test_Resource_Namespace_GetDesiredState(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		Obj          interface{}
		ExpectedName string
	}{
		{
			Obj: &v2alpha1.AzureConfig{
				Spec: v2alpha1.AzureConfigSpec{
					Cluster: v2alpha1.Cluster{
						ID: "al9qy",
					},
				},
			},
			ExpectedName: "al9qy",
		},
		{
			Obj: &v2alpha1.AzureConfig{
				Spec: v2alpha1.AzureConfigSpec{
					Cluster: v2alpha1.Cluster{
						ID: "foobar",
					},
				},
			},
			ExpectedName: "foobar",
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
		result, err := newResource.GetDesiredState(context.TODO(), tc.Obj)
		if err != nil {
			t.Fatal("case", i+1, "expected", nil, "got", err)
		}
		name := result.(*corev2.Namespace).Name
		if tc.ExpectedName != name {
			t.Fatalf("case %d expected %#v got %#v", i+1, tc.ExpectedName, name)
		}
	}
}
