package namespace

import (
	"context"
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/giantswarm/apiextensions/v6/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/micrologger/microloggertest"

	"github.com/giantswarm/azure-operator/v6/pkg/label"
	"github.com/giantswarm/azure-operator/v6/service/controller/key"
)

func Test_Resource_Namespace_GetDesiredState(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		Obj            interface{}
		ExpectedName   string
		ExpectedLabels map[string]string
	}{
		{
			Obj: &v1alpha1.AzureConfig{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						label.Cluster:         "al9qy",
						label.OperatorVersion: "0.1.0",
					},
				},
				Spec: v1alpha1.AzureConfigSpec{
					Cluster: v1alpha1.Cluster{
						ID: "al9qy",
					},
				},
			},
			ExpectedName: "al9qy",
			ExpectedLabels: map[string]string{
				key.LabelApp:           "master",
				key.LegacyLabelCluster: "al9qy",
				key.LabelCustomer:      "",
				key.LabelCluster:       "al9qy",
				key.LabelOrganization:  "",
			},
		},
		{
			Obj: &v1alpha1.AzureConfig{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						label.Cluster:         "foobar",
						label.OperatorVersion: "0.1.0",
					},
				},
				Spec: v1alpha1.AzureConfigSpec{
					Cluster: v1alpha1.Cluster{
						ID: "foobar",
					},
				},
			},
			ExpectedName: "foobar",
			ExpectedLabels: map[string]string{
				key.LabelApp:           "master",
				key.LegacyLabelCluster: "foobar",
				key.LabelCustomer:      "",
				key.LabelCluster:       "foobar",
				key.LabelOrganization:  "",
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
		result, err := newResource.GetDesiredState(context.TODO(), tc.Obj)
		if err != nil {
			t.Fatal("case", i+1, "expected", nil, "got", err)
		}
		namespace := result.(*corev1.Namespace)
		if tc.ExpectedName != namespace.Name {
			t.Fatalf("case %d expected %#v got %#v", i+1, tc.ExpectedName, namespace.Name)
		}

		if !reflect.DeepEqual(tc.ExpectedLabels, namespace.GetLabels()) {
			t.Fatalf("case %d expected labels %#v got %#v", i+1, tc.ExpectedLabels, namespace.GetLabels())
		}
	}
}
