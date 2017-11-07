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
	"github.com/giantswarm/operatorkit/framework"
	"github.com/spf13/viper"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/flag"
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
		cloudConfigConfig.Flag = flag.New()
		cloudConfigConfig.Logger = microloggertest.New()
		cloudConfigConfig.Viper = viper.New()

		cloudConfigService, err := cloudconfig.New(cloudConfigConfig)
		if err != nil {
			t.Fatalf("expected '%v' got '%#v'", nil, err)
		}

		resourceConfig := DefaultConfig()
		resourceConfig.AzureConfig = client.DefaultAzureConfig()
		resourceConfig.CertWatcher = certificatetprtest.NewService()
		resourceConfig.CloudConfig = cloudConfigService
		resourceConfig.Logger = microloggertest.New()
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
func Test_Resource_Deployment_NewUpdatePatch(t *testing.T) {
	testCases := []struct {
		Obj               interface{}
		Cur               interface{}
		Des               interface{}
		ExpectedPatchFunc func(*framework.Patch)
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
			ExpectedPatchFunc: func(patch *framework.Patch) {
				deploymentsToCreate := []Deployment{
					Deployment{
						Name: "cluster-setup",
					},
				}

				patch.SetCreateChange(deploymentsToCreate)
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
			ExpectedPatchFunc: func(patch *framework.Patch) {},
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
			ExpectedPatchFunc: func(patch *framework.Patch) {
				deploymentsToCreate := []Deployment{
					Deployment{
						Name: "network-setup",
					},
				}

				patch.SetCreateChange(deploymentsToCreate)
			},
		},
	}

	var newResource *Resource
	{
		cloudConfigConfig := cloudconfig.DefaultConfig()
		cloudConfigConfig.Flag = flag.New()
		cloudConfigConfig.Logger = microloggertest.New()
		cloudConfigConfig.Viper = viper.New()

		cloudConfigService, err := cloudconfig.New(cloudConfigConfig)
		if err != nil {
			t.Fatalf("expected '%#v' got '%#v'", nil, err)
		}

		resourceConfig := DefaultConfig()
		resourceConfig.AzureConfig = client.DefaultAzureConfig()
		resourceConfig.CertWatcher = certificatetprtest.NewService()
		resourceConfig.CloudConfig = cloudConfigService
		resourceConfig.Logger = microloggertest.New()
		newResource, err = New(resourceConfig)
		if err != nil {
			t.Fatalf("expected '%#v' got '%#v'", nil, err)
		}
	}
	for i, tc := range testCases {
		patch, err := newResource.NewUpdatePatch(context.TODO(), tc.Obj, tc.Cur, tc.Des)
		if err != nil {
			t.Fatalf("case %d expected %#v got %#v", i+1, nil, err)
		}

		expectedPatch := framework.NewPatch()
		tc.ExpectedPatchFunc(expectedPatch)

		if !reflect.DeepEqual(patch, expectedPatch) {
			t.Fatalf("case %d expected %#v got %#v", i+1, expectedPatch, patch)
		}
	}
}
