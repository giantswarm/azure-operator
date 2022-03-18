package namespace

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	apismetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/giantswarm/apiextensions/v5/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/micrologger/microloggertest"
)

func Test_Resource_Namespace_newDeleteChange(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		Obj               interface{}
		Cur               interface{}
		Des               interface{}
		ExpectedNamespace *corev1.Namespace
	}{
		{
			Obj: &v1alpha1.AzureConfig{
				Spec: v1alpha1.AzureConfigSpec{
					Cluster: v1alpha1.Cluster{
						ID: "foobar",
					},
				},
			},
			Cur: &corev1.Namespace{
				TypeMeta: apismetav1.TypeMeta{
					Kind:       "Namespace",
					APIVersion: "v1",
				},
				ObjectMeta: apismetav1.ObjectMeta{
					Name: "al9qy",
					Labels: map[string]string{
						"cluster":  "al9qy",
						"customer": "test-customer",
					},
				},
			},
			Des: &corev1.Namespace{
				TypeMeta: apismetav1.TypeMeta{
					Kind:       "Namespace",
					APIVersion: "v1",
				},
				ObjectMeta: apismetav1.ObjectMeta{
					Name: "al9qy",
					Labels: map[string]string{
						"cluster":  "al9qy",
						"customer": "test-customer",
					},
				},
			},
			ExpectedNamespace: &corev1.Namespace{
				TypeMeta: apismetav1.TypeMeta{
					Kind:       "Namespace",
					APIVersion: "v1",
				},
				ObjectMeta: apismetav1.ObjectMeta{
					Name: "al9qy",
					Labels: map[string]string{
						"cluster":  "al9qy",
						"customer": "test-customer",
					},
				},
			},
		},

		{
			Obj: &v1alpha1.AzureConfig{
				Spec: v1alpha1.AzureConfigSpec{
					Cluster: v1alpha1.Cluster{
						ID: "foobar",
					},
				},
			},
			Cur: nil,
			Des: &corev1.Namespace{
				TypeMeta: apismetav1.TypeMeta{
					Kind:       "Namespace",
					APIVersion: "v1",
				},
				ObjectMeta: apismetav1.ObjectMeta{
					Name: "al9qy",
					Labels: map[string]string{
						"cluster":  "al9qy",
						"customer": "test-customer",
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
		result, err := newResource.newDeleteChange(context.TODO(), tc.Cur, tc.Des)
		if err != nil {
			t.Fatal("case", i+1, "expected", nil, "got", err)
		}
		if tc.ExpectedNamespace == nil {
			if tc.ExpectedNamespace != result {
				t.Fatal("case", i+1, "expected", tc.ExpectedNamespace, "got", result)
			}
		} else {
			name := result.(*corev1.Namespace).Name
			if tc.ExpectedNamespace.Name != name {
				t.Fatal("case", i+1, "expected", tc.ExpectedNamespace.Name, "got", name)
			}
		}
	}
}
