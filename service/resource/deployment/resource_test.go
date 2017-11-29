package deployment

import (
	"context"
	"reflect"
	"testing"

	"github.com/giantswarm/azuretpr"
	"github.com/giantswarm/certificatetpr/certificatetprtest"
	"github.com/giantswarm/clustertpr"
	"github.com/giantswarm/clustertpr/spec"
	"github.com/giantswarm/micrologger/microloggertest"

	"github.com/giantswarm/azure-operator/client/fakeclient"
	"github.com/giantswarm/azure-operator/service/cloudconfig"
)

func Test_Resource_Deployment_GetDesiredState(t *testing.T) {
	customObject := &azuretpr.CustomObject{
		Spec: azuretpr.Spec{
			Cluster: clustertpr.Spec{
				Cluster: spec.Cluster{
					ID: "5xchu",
				},
			},
		},
	}
	expectedDeployments := []Deployment{
		Deployment{
			Name:          "cluster-main-template",
			ResourceGroup: "5xchu",
		},
	}

	var err error
	var newResource *Resource
	{
		cloudConfigConfig := cloudconfig.DefaultConfig()
		cloudConfigConfig.AzureConfig = fakeclient.NewAzureConfig()
		cloudConfigConfig.Logger = microloggertest.New()

		cloudConfigService, err := cloudconfig.New(cloudConfigConfig)
		if err != nil {
			t.Fatalf("expected '%v' got '%#v'", nil, err)
		}

		resourceConfig := DefaultConfig()
		resourceConfig.AzureConfig = fakeclient.NewAzureConfig()
		resourceConfig.CertWatcher = certificatetprtest.NewService()
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

	deployments, ok := result.([]Deployment)
	if !ok {
		t.Fatalf("case expected '%T', got '%T'", []Deployment{}, deployments)
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
		ExpectedDeployments []Deployment
	}{
		{
			// Case 1. Current state is empty. A deployment is created.
			Obj: &azuretpr.CustomObject{
				Spec: azuretpr.Spec{
					Cluster: clustertpr.Spec{
						Cluster: spec.Cluster{
							ID: "5xchu",
						},
					},
				},
			},
			Cur: []Deployment{},
			Des: []Deployment{
				Deployment{
					Name: "cluster-setup",
				},
			},
			ExpectedDeployments: []Deployment{
				Deployment{
					Name: "cluster-setup",
				},
			},
		},
		{
			// Case 2. Current and desired states are the same.
			// No deployments are created.
			Obj: &azuretpr.CustomObject{
				Spec: azuretpr.Spec{
					Cluster: clustertpr.Spec{
						Cluster: spec.Cluster{
							ID: "5xchu",
						},
					},
				},
			},
			Cur: []Deployment{
				Deployment{
					Name: "cluster-setup",
				},
			},
			Des: []Deployment{
				Deployment{
					Name: "cluster-setup",
				},
			},
			ExpectedDeployments: nil,
		},
		{
			// Case 3. Current and desired states are different.
			// A new deployment is created.
			Obj: &azuretpr.CustomObject{
				Spec: azuretpr.Spec{
					Cluster: clustertpr.Spec{
						Cluster: spec.Cluster{
							ID: "5xchu",
						},
					},
				},
			},
			Cur: []Deployment{
				Deployment{
					Name: "cluster-setup",
				},
			},
			Des: []Deployment{
				Deployment{
					Name: "cluster-setup",
				},
				Deployment{
					Name: "network-setup",
				},
			},
			ExpectedDeployments: []Deployment{
				Deployment{
					Name: "network-setup",
				},
			},
		},
	}

	var newResource *Resource
	{
		cloudConfigConfig := cloudconfig.DefaultConfig()
		cloudConfigConfig.AzureConfig = fakeclient.NewAzureConfig()
		cloudConfigConfig.Logger = microloggertest.New()

		cloudConfigService, err := cloudconfig.New(cloudConfigConfig)
		if err != nil {
			t.Fatalf("expected '%#v' got '%#v'", nil, err)
		}

		resourceConfig := DefaultConfig()
		resourceConfig.AzureConfig = fakeclient.NewAzureConfig()
		resourceConfig.CertWatcher = certificatetprtest.NewService()
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
