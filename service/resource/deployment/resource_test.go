package deployment

import (
	"context"
	"testing"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azuretpr"
	"github.com/giantswarm/clustertpr"
	"github.com/giantswarm/clustertpr/spec"
	"github.com/giantswarm/micrologger/microloggertest"
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
		resourceConfig := DefaultConfig()
		resourceConfig.AzureConfig = client.DefaultAzureConfig()
		resourceConfig.Logger = microloggertest.New()
		newResource, err = New(resourceConfig)
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
func Test_Resource_Deployment_GetCreateState(t *testing.T) {
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
			ExpectedDeployments: []Deployment{},
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

	var err error
	var newResource *Resource
	{
		resourceConfig := DefaultConfig()
		resourceConfig.AzureConfig = client.DefaultAzureConfig()
		resourceConfig.Logger = microloggertest.New()
		newResource, err = New(resourceConfig)
		if err != nil {
			t.Fatalf("expected '%#v' got '%#v'", nil, err)
		}
	}
	for i, tc := range testCases {
		result, err := newResource.GetCreateState(context.TODO(), tc.Obj, tc.Cur, tc.Des)
		if err != nil {
			t.Fatalf("case %d expected '%v' got '%#v'", i+1, nil, err)
		}

		deployments, ok := result.([]Deployment)
		if !ok {
			t.Fatalf("case %d expected '%T', got '%T'", i+1, []Deployment{}, deployments)
		}

		if len(tc.ExpectedDeployments) != len(deployments) {
			t.Fatalf("case %d expected %d deployments got %d", i+1, len(tc.ExpectedDeployments), len(deployments))
		}
	}
}

func Test_Resource_Deployment_GetDeleteState(t *testing.T) {
	testCases := []struct {
		Obj                 interface{}
		Cur                 interface{}
		Des                 interface{}
		ExpectedDeployments []Deployment
	}{
		{
			// Case 1. Current and desired states are the same.
			// No deployments are returned because deployments aren't deleted.
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
			ExpectedDeployments: []Deployment{},
		},
		{
			// Case 2. Current state is empty.
			// No deployments are returned because deployments aren't deleted.
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
			ExpectedDeployments: []Deployment{},
		},
	}

	var err error
	var newResource *Resource
	{
		resourceConfig := DefaultConfig()
		resourceConfig.AzureConfig = client.DefaultAzureConfig()
		resourceConfig.Logger = microloggertest.New()
		newResource, err = New(resourceConfig)
		if err != nil {
			t.Fatalf("expected '%#v' got '%#v'", nil, err)
		}
	}
	for i, tc := range testCases {
		result, err := newResource.GetDeleteState(context.TODO(), tc.Obj, tc.Cur, tc.Des)
		if err != nil {
			t.Fatalf("case %d expected '%v' got '%#v'", i+1, nil, err)
		}

		deployments, ok := result.([]Deployment)
		if !ok {
			t.Fatalf("case %d expected '%T', got '%T'", i+1, []Deployment{}, deployments)
		}

		if len(deployments) != 0 {
			t.Fatalf("case %d expected 0 deployments got %d", i+1, len(deployments))
		}
	}
}
