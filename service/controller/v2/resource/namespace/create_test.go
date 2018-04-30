package namespace

import (
	"context"
	"testing"

	corev2 "k8s.io/api/core/v1"
	apismetav2 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/giantswarm/apiextensions/pkg/apis/provider/v2alpha1"
	"github.com/giantswarm/micrologger/microloggertest"
)

func Test_Resource_Namespace_newCreateChange(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		Obj               interface{}
		Cur               interface{}
		Des               interface{}
		ExpectedNamespace *corev2.Namespace
	}{
		{
			Obj: &v2alpha1.AzureConfig{
				Spec: v2alpha1.AzureConfigSpec{
					Cluster: v2alpha1.Cluster{
						ID: "foobar",
					},
				},
			},
			Cur: &corev2.Namespace{
				TypeMeta: apismetav2.TypeMeta{
					Kind:       "Namespace",
					APIVersion: "v2",
				},
				ObjectMeta: apismetav2.ObjectMeta{
					Name: "al9qy",
					Labels: map[string]string{
						"cluster":  "al9qy",
						"customer": "test-customer",
					},
				},
			},
			Des: &corev2.Namespace{
				TypeMeta: apismetav2.TypeMeta{
					Kind:       "Namespace",
					APIVersion: "v2",
				},
				ObjectMeta: apismetav2.ObjectMeta{
					Name: "al9qy",
					Labels: map[string]string{
						"cluster":  "al9qy",
						"customer": "test-customer",
					},
				},
			},
			ExpectedNamespace: nil,
		},

		{
			Obj: &v2alpha1.AzureConfig{
				Spec: v2alpha1.AzureConfigSpec{
					Cluster: v2alpha1.Cluster{
						ID: "foobar",
					},
				},
			},
			Cur: nil,
			Des: &corev2.Namespace{
				TypeMeta: apismetav2.TypeMeta{
					Kind:       "Namespace",
					APIVersion: "v2",
				},
				ObjectMeta: apismetav2.ObjectMeta{
					Name: "al9qy",
					Labels: map[string]string{
						"cluster":  "al9qy",
						"customer": "test-customer",
					},
				},
			},
			ExpectedNamespace: &corev2.Namespace{
				TypeMeta: apismetav2.TypeMeta{
					Kind:       "Namespace",
					APIVersion: "v2",
				},
				ObjectMeta: apismetav2.ObjectMeta{
					Name: "al9qy",
					Labels: map[string]string{
						"cluster":  "al9qy",
						"customer": "test-customer",
					},
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

	for i, tc := range testCases {
		result, err := newResource.newCreateChange(context.TODO(), tc.Obj, tc.Cur, tc.Des)
		if err != nil {
			t.Fatal("case", i+1, "expected", nil, "got", err)
		}
		if tc.ExpectedNamespace == nil {
			if tc.ExpectedNamespace != result {
				t.Fatal("case", i+1, "expected", tc.ExpectedNamespace, "got", result)
			}
		} else {
			name := result.(*corev2.Namespace).Name
			if tc.ExpectedNamespace.Name != name {
				t.Fatal("case", i+1, "expected", tc.ExpectedNamespace.Name, "got", name)
			}
		}
	}
}
