package release

import (
	"context"
	"reflect"
	"testing"
	"time"

	providerv1alpha1 "github.com/giantswarm/apiextensions/v2/pkg/apis/provider/v1alpha1"
	releasev1alpha1 "github.com/giantswarm/apiextensions/v2/pkg/apis/release/v1alpha1"
	"github.com/giantswarm/micrologger/microloggertest"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/azure-operator/v4/pkg/project"
	"github.com/giantswarm/azure-operator/v4/service/controller/controllercontext"
	"github.com/giantswarm/azure-operator/v4/service/unittest"
)

func Test_Resource_Puts_Release_In_Context_When_Release_Has_Leading_V(t *testing.T) {
	ctx := context.Background()
	ctx = controllercontext.NewContext(ctx, controllercontext.Context{})

	client := unittest.FakeK8sClient()

	config := Config{
		K8sClient: client,
		Logger:    microloggertest.New(),
	}
	resource, err := New(config)
	if err != nil {
		t.Fatal(err)
	}

	release := givenReleaseWithName("v1.0.0")
	azureConfig := givenAzureConfigWithReleaseLabel("v1.0.0")

	err = client.CtrlClient().Create(ctx, release)
	if err != nil {
		t.Fatal(err)
	}

	err = resource.EnsureCreated(ctx, azureConfig)
	if err != nil {
		t.Fatal(err)
	}

	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if reflect.DeepEqual(cc.Release.Release, release) {
		t.Fatalf("The release is not in the context")
	}
}

func Test_Resource_Puts_Release_In_Context_When_Release_Has_Not_Leading_V(t *testing.T) {
	ctx := context.Background()
	ctx = controllercontext.NewContext(ctx, controllercontext.Context{})

	client := unittest.FakeK8sClient()

	config := Config{
		K8sClient: client,
		Logger:    microloggertest.New(),
	}
	resource, err := New(config)
	if err != nil {
		t.Fatal(err)
	}

	release := givenReleaseWithName("v1.0.0")
	azureConfig := givenAzureConfigWithReleaseLabel("1.0.0")

	err = client.CtrlClient().Create(ctx, release)
	if err != nil {
		t.Fatal(err)
	}

	err = resource.EnsureCreated(ctx, azureConfig)
	if err != nil {
		t.Fatal(err)
	}

	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if reflect.DeepEqual(cc.Release.Release, release) {
		t.Fatalf("The release is not in the context")
	}
}

func givenReleaseWithName(releaseName string) *releasev1alpha1.Release {
	return &releasev1alpha1.Release{
		ObjectMeta: metav1.ObjectMeta{
			Name: releaseName,
			Labels: map[string]string{
				"giantswarm.io/managed-by": "release-operator",
				"giantswarm.io/provider":   "azure",
			},
		},
		Spec: releasev1alpha1.ReleaseSpec{
			Apps: []releasev1alpha1.ReleaseSpecApp{},
			Components: []releasev1alpha1.ReleaseSpecComponent{
				{
					Name:    project.Name(),
					Version: project.Version(),
				},
				{
					Name:    "cluster-operator",
					Version: "0.23.8",
				},
				{
					Name:    "cert-operator",
					Version: "0.1.0",
				},
				{
					Name:    "app-operator",
					Version: "1.0.0",
				},
				{
					Name:    "calico",
					Version: "3.10.1",
				},
				{
					Name:    "containerlinux",
					Version: "2345.3.1",
				},
				{
					Name:    "coredns",
					Version: "1.6.5",
				},
				{
					Name:    "etcd",
					Version: "3.3.17",
				},
				{
					Name:    "kubernetes",
					Version: "1.16.8",
				},
			},
			Date:  &metav1.Time{Time: time.Unix(10, 0)},
			State: "active",
		},
	}
}

func givenAzureConfigWithReleaseLabel(releaseName string) *providerv1alpha1.AzureConfig {
	clusterName := "test-cluster"
	return &providerv1alpha1.AzureConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterName,
			Namespace: "default",
			Labels: map[string]string{
				"giantswarm.io/cluster":                clusterName,
				"azure-operator.giantswarm.io/version": "x.y.z",
				"release.giantswarm.io/version":        releaseName,
			},
		},
		Spec: providerv1alpha1.AzureConfigSpec{},
	}
}
