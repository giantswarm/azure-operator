package deployment

import (
	"context"
	"reflect"
	"testing"

	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/certs/certstest"
	"github.com/giantswarm/micrologger/microloggertest"
	"github.com/giantswarm/randomkeys/randomkeystest"

	"github.com/giantswarm/azure-operator/client/fakeclient"
	"github.com/giantswarm/azure-operator/service/cloudconfig"
)

func Test_Resource_Deployment_GetDesiredState(t *testing.T) {
	customObject := &providerv1alpha1.AzureConfig{
		Spec: providerv1alpha1.AzureConfigSpec{
			Cluster: providerv1alpha1.Cluster{
				ID: "5xchu",
			},
		},
	}
	expectedDeployments := []deployment{
		deployment{
			Name:          "cluster-main-template",
			ResourceGroup: "5xchu",
		},
	}

	var err error
	var newResource *Resource
	{
		cloudConfigConfig := cloudconfig.Config{
			CertsSearcher:      certstest.NewSearcher(),
			Logger:             microloggertest.New(),
			RandomkeysSearcher: randomkeystest.NewSearcher(),

			AzureConfig: fakeclient.NewAzureConfig(),
		}

		cloudConfigService, err := cloudconfig.New(cloudConfigConfig)
		if err != nil {
			t.Fatalf("expected '%v' got '%#v'", nil, err)
		}

		resourceConfig := DefaultConfig()
		resourceConfig.AzureConfig = fakeclient.NewAzureConfig()
		resourceConfig.CertsSearcher = certstest.NewSearcher()
		resourceConfig.CloudConfig = cloudConfigService
		resourceConfig.Logger = microloggertest.New()
		resourceConfig.TemplateVersion = "master"
		newResource, err = New(resourceConfig)
		if err != nil {
			t.Fatalf("expected '%v' got '%#v'", nil, err)
		}
	}

	result, err := newResource.GetDesiredState(context.TODO(), customObject)
	if err != nil {
		t.Fatalf("expected '%v' got '%#v'", nil, err)
	}

	deployments, ok := result.([]deployment)
	if !ok {
		t.Fatalf("case expected '%T', got '%T'", []deployment{}, deployments)
	}

	if len(expectedDeployments) != len(deployments) {
		t.Fatalf("expected %d deployments got %d", len(expectedDeployments), len(deployments))
	}
	if expectedDeployments[0].Name != deployments[0].Name {
		t.Fatalf("expected deployment name '%s' got '%s'", expectedDeployments[0].Name, deployments[0].Name)
	}
	if expectedDeployments[0].ResourceGroup != deployments[0].ResourceGroup {
		t.Fatalf("expected deployment name '%s' got '%s'", expectedDeployments[0].ResourceGroup, deployments[0].ResourceGroup)
	}
}

func Test_Resource_Deployment_newCreateChange(t *testing.T) {
	testCases := []struct {
		Obj                 interface{}
		Cur                 interface{}
		Des                 interface{}
		ExpectedDeployments []deployment
	}{
		{
			// Case 1. Current state is empty. A deployment is created.
			Obj: &providerv1alpha1.AzureConfig{
				Spec: providerv1alpha1.AzureConfigSpec{
					Cluster: providerv1alpha1.Cluster{
						ID: "5xchu",
					},
				},
			},
			Cur: []deployment{},
			Des: []deployment{
				deployment{
					Name: "cluster-setup",
				},
			},
			ExpectedDeployments: []deployment{
				deployment{
					Name: "cluster-setup",
				},
			},
		},
		{
			// Case 2. Current and desired states are the same.
			// No deployments are created.
			Obj: &providerv1alpha1.AzureConfig{
				Spec: providerv1alpha1.AzureConfigSpec{
					Cluster: providerv1alpha1.Cluster{
						ID: "5xchu",
					},
				},
			},
			Cur: []deployment{
				deployment{
					Name: "cluster-setup",
				},
			},
			Des: []deployment{
				deployment{
					Name: "cluster-setup",
				},
			},
			ExpectedDeployments: nil,
		},
		{
			// Case 3. Current and desired states are different.
			// A new deployment is created.
			Obj: &providerv1alpha1.AzureConfig{
				Spec: providerv1alpha1.AzureConfigSpec{
					Cluster: providerv1alpha1.Cluster{
						ID: "5xchu",
					},
				},
			},
			Cur: []deployment{
				deployment{
					Name: "cluster-setup",
				},
			},
			Des: []deployment{
				deployment{
					Name: "cluster-setup",
				},
				deployment{
					Name: "network-setup",
				},
			},
			ExpectedDeployments: []deployment{
				deployment{
					Name: "network-setup",
				},
			},
		},
	}

	var newResource *Resource
	{
		cloudConfigConfig := cloudconfig.Config{
			CertsSearcher:      certstest.NewSearcher(),
			Logger:             microloggertest.New(),
			RandomkeysSearcher: randomkeystest.NewSearcher(),

			AzureConfig: fakeclient.NewAzureConfig(),
		}

		cloudConfigService, err := cloudconfig.New(cloudConfigConfig)
		if err != nil {
			t.Fatalf("expected '%#v' got '%#v'", nil, err)
		}

		resourceConfig := DefaultConfig()
		resourceConfig.AzureConfig = fakeclient.NewAzureConfig()
		resourceConfig.CertsSearcher = certstest.NewSearcher()
		resourceConfig.CloudConfig = cloudConfigService
		resourceConfig.Logger = microloggertest.New()
		resourceConfig.TemplateVersion = "master"
		newResource, err = New(resourceConfig)
		if err != nil {
			t.Fatalf("expected '%#v' got '%#v'", nil, err)
		}
	}
	for i, tc := range testCases {
		deployments, err := newResource.newCreateChange(context.TODO(), tc.Obj, tc.Cur, tc.Des)
		if err != nil {
			t.Fatalf("case %d expected %#v got %#v", i+1, nil, err)
		}

		//if !reflect.DeepEqual(deployments, tc.ExpectedDeployments) {
		if !reflect.DeepEqual(deployments, tc.ExpectedDeployments) {
			t.Fatalf("case %d expected %#v got %#v", i+1, tc.ExpectedDeployments, deployments)
		}
	}
}
