package resourcegroup

import (
	"context"
	"reflect"
	"testing"

	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/clustertpr/spec"
	"github.com/giantswarm/micrologger/microloggertest"

	"github.com/giantswarm/azure-operator/client/fakeclient"
)

func Test_Resource_ResourceGroup_GetDesiredState(t *testing.T) {
	testCases := []struct {
		Obj              interface{}
		ExpectedName     string
		ExpectedLocation string
		ExpectedTags     map[string]string
	}{
		{
			// Case 1. Standard cluster ID format.
			Obj: &providerv1alpha1.AzureConfig{
				Spec: providerv1alpha1.AzureConfigSpec{
					Azure: providerv1alpha1.AzureConfigSpecAzure{
						Location: "West Europe",
					},
					Cluster: providerv1alpha1.Cluster{
						ID: "5xchu",
						Customer: providerv1alpha1.ClusterCustomer{
							ID: "giantswarm",
						},
					},
				},
			},
			ExpectedName:     "5xchu",
			ExpectedLocation: "West Europe",
			ExpectedTags: map[string]string{
				"ClusterID":  "5xchu",
				"CustomerID": "giantswarm",
			},
		},
		{
			// Case 2. Custom cluster ID format.
			Obj: &providerv1alpha1.AzureConfig{
				Spec: providerv1alpha1.AzureConfigSpec{
					Azure: providerv1alpha1.AzureConfigSpecAzure{
						Location: "East Asia",
					},
					Cluster: providerv1alpha1.Spec{
						Cluster: spec.Cluster{
							ID: "test-cluster",
						},
						Customer: spec.Customer{
							ID: "acme",
						},
					},
				},
			},
			ExpectedName:     "test-cluster",
			ExpectedLocation: "East Asia",
			ExpectedTags: map[string]string{
				"ClusterID":  "test-cluster",
				"CustomerID": "acme",
			},
		},
	}

	var err error
	var newResource *Resource
	{
		resourceConfig := DefaultConfig()
		resourceConfig.AzureConfig = fakeclient.NewAzureConfig()
		resourceConfig.Logger = microloggertest.New()
		newResource, err = New(resourceConfig)
		if err != nil {
			t.Fatalf("expected '%#v' got '%#v'", nil, err)
		}

		for i, tc := range testCases {
			result, err := newResource.GetDesiredState(context.TODO(), tc.Obj)
			if err != nil {
				t.Fatalf("case %d expected '%v' got '%#v'", i+1, nil, err)
			}

			group, ok := result.(Group)
			if !ok {
				t.Fatalf("case %d expected '%T', got '%T'", i+1, Group{}, group)
			}
			if tc.ExpectedName != group.Name {
				t.Fatalf("case %d expected name '%s' got '%s'", i+1, tc.ExpectedName, group.Name)
			}
			if tc.ExpectedLocation != group.Location {
				t.Fatalf("case %d expected location '%s' got '%s'", i+1, tc.ExpectedLocation, group.Location)
			}

			if len(tc.ExpectedTags) != len(group.Tags) {
				t.Fatalf("case %d expected %d tags got %d", i+1, len(tc.ExpectedTags), len(group.Tags))
			}

			for tag, val := range group.Tags {
				expectedVal, ok := tc.ExpectedTags[tag]
				if !ok {
					t.Fatalf("case %d tag '%s' was not expected", i+1, tag)
				}

				if val != expectedVal {
					t.Fatalf("case %d expected value '%s' for tag '%s' got '%s'", i+1, expectedVal, tag, val)
				}
			}
		}
	}
}

func Test_Resource_ResourceGroup_newCreateChange(t *testing.T) {
	testCases := []struct {
		Obj                   interface{}
		Cur                   interface{}
		Des                   interface{}
		ExpectedResourceGroup Group
	}{
		{
			// Case 1. Current and desired states are the same. The resource
			// group should not be created.
			Obj: &providerv1alpha1.AzureConfig{
				Spec: providerv1alpha1.AzureConfigSpec{
					Cluster: providerv1alpha1.Cluster{
						ID: "5xchu",
					},
				},
			},
			Cur: Group{
				Name: "5xchu",
			},
			Des: Group{
				Name: "5xchu",
			},
			ExpectedResourceGroup: Group{},
		},
		{
			// Case 2. Current state is nil. The resource group should be
			// created.
			Obj: &providerv1alpha1.AzureConfig{
				Spec: providerv1alpha1.AzureConfigSpec{
					Cluster: providerv1alpha1.Cluster{
						ID: "5xchu",
					},
				},
			},
			Cur: Group{},
			Des: Group{
				Name: "5xchu",
			},
			ExpectedResourceGroup: Group{
				Name: "5xchu",
			},
		},
	}

	var err error
	var newResource *Resource
	{
		resourceConfig := DefaultConfig()
		resourceConfig.AzureConfig = fakeclient.NewAzureConfig()
		resourceConfig.Logger = microloggertest.New()
		newResource, err = New(resourceConfig)
		if err != nil {
			t.Fatalf("expected '%#v' got '%#v'", nil, err)
		}
	}

	for i, tc := range testCases {
		resourceGroup, err := newResource.newCreateChange(context.TODO(), tc.Obj, tc.Cur, tc.Des)
		if err != nil {
			t.Fatalf("case %d expected '%v' got '%#v'", i+1, nil, err)
		}

		if !reflect.DeepEqual(resourceGroup, tc.ExpectedResourceGroup) {
			t.Fatalf("case %d expected %#v, got %#v", i+1, resourceGroup, tc.ExpectedResourceGroup)
		}
	}
}

func Test_Resource_ResourceGroup_newDeleteChange(t *testing.T) {
	testCases := []struct {
		Obj                   interface{}
		Cur                   interface{}
		Des                   interface{}
		ExpectedResourceGroup Group
	}{
		{
			// Case 1. Current and desired states are the same. The resource
			// group should be deleted.
			Obj: &providerv1alpha1.AzureConfig{
				Spec: providerv1alpha1.AzureConfigSpec{
					Cluster: providerv1alpha1.Cluster{
						ID: "5xchu",
					},
				},
			},
			Cur: Group{
				Name: "5xchu",
			},
			Des: Group{
				Name: "5xchu",
			},
			ExpectedResourceGroup: Group{
				Name: "5xchu",
			},
		},
		{
			// Case 2. Current state is nil. The resource group should not be
			// deleted.
			Obj: &providerv1alpha1.AzureConfig{
				Spec: providerv1alpha1.AzureConfigSpec{
					Cluster: providerv1alpha1.Cluster{
						ID: "5xchu",
					},
				},
			},
			Cur: Group{},
			Des: Group{
				Name: "5xchu",
			},
			ExpectedResourceGroup: Group{},
		},
	}

	var err error
	var newResource *Resource
	{
		resourceConfig := DefaultConfig()
		resourceConfig.AzureConfig = fakeclient.NewAzureConfig()
		resourceConfig.Logger = microloggertest.New()
		newResource, err = New(resourceConfig)
		if err != nil {
			t.Fatalf("expected '%#v' got '%#v'", nil, err)
		}
	}

	for i, tc := range testCases {
		resourceGroup, err := newResource.newDeleteChange(context.TODO(), tc.Obj, tc.Cur, tc.Des)
		if err != nil {
			t.Fatalf("case %d expected '%v' got '%#v'", i+1, nil, err)
		}

		if !reflect.DeepEqual(resourceGroup, tc.ExpectedResourceGroup) {
			t.Fatalf("case %d expected %#v, got %#v", i+1, resourceGroup, tc.ExpectedResourceGroup)
		}
	}
}
