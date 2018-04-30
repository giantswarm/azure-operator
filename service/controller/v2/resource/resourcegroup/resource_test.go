package resourcegroup

import (
	"context"
	"reflect"
	"testing"

	providerv2alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/micrologger/microloggertest"

	"github.com/giantswarm/azure-operator/client/fakeclient"
	"github.com/giantswarm/azure-operator/service/controller/setting"
)

var testAzure = setting.Azure{
	HostCluster: setting.AzureHostCluster{
		CIDR:           "10.0.0.0/8",
		ResourceGroup:  "test-group",
		VirtualNetwork: "test-vnet",
	},
	Location: "westeurope",
}

func Test_Resource_ResourceGroup_GetDesiredState(t *testing.T) {
	testCases := []struct {
		Obj              interface{}
		InstallationName string
		ExpectedName     string
		ExpectedTags     map[string]string
	}{
		{
			// Case 1. Standard cluster ID format.
			Obj: &providerv2alpha1.AzureConfig{
				Spec: providerv2alpha1.AzureConfigSpec{
					Cluster: providerv2alpha1.Cluster{
						ID: "5xchu",
						Customer: providerv2alpha1.ClusterCustomer{
							ID: "giantswarm",
						},
					},
				},
			},
			InstallationName: "gollum",
			ExpectedName:     "5xchu",
			ExpectedTags: map[string]string{
				"GiantSwarmCluster":      "5xchu",
				"GiantSwarmInstallation": "gollum",
				"GiantSwarmOrganization": "giantswarm",
			},
		},
		{
			// Case 2. Custom cluster ID format.
			Obj: &providerv2alpha1.AzureConfig{
				Spec: providerv2alpha1.AzureConfigSpec{
					Cluster: providerv2alpha1.Cluster{
						ID: "test-cluster",
						Customer: providerv2alpha1.ClusterCustomer{
							ID: "acme",
						},
					},
				},
			},
			InstallationName: "coyote",
			ExpectedName:     "test-cluster",
			ExpectedTags: map[string]string{
				"GiantSwarmCluster":      "test-cluster",
				"GiantSwarmInstallation": "coyote",
				"GiantSwarmOrganization": "acme",
			},
		},
	}

	var err error

	for i, tc := range testCases {
		var resource *Resource
		{

			c := Config{
				Logger: microloggertest.New(),

				Azure:            testAzure,
				AzureConfig:      fakeclient.NewAzureConfig(),
				InstallationName: tc.InstallationName,
			}

			resource, err = New(c)
			if err != nil {
				t.Fatalf("expected '%#v' got '%#v'", nil, err)
			}
		}

		result, err := resource.GetDesiredState(context.TODO(), tc.Obj)
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
			Obj: &providerv2alpha1.AzureConfig{
				Spec: providerv2alpha1.AzureConfigSpec{
					Cluster: providerv2alpha1.Cluster{
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
			Obj: &providerv2alpha1.AzureConfig{
				Spec: providerv2alpha1.AzureConfigSpec{
					Cluster: providerv2alpha1.Cluster{
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

	var resource *Resource
	{
		c := Config{
			Logger: microloggertest.New(),

			Azure:            testAzure,
			AzureConfig:      fakeclient.NewAzureConfig(),
			InstallationName: "test-installation",
		}

		resource, err = New(c)
		if err != nil {
			t.Fatalf("expected '%#v' got '%#v'", nil, err)
		}
	}

	for i, tc := range testCases {
		resourceGroup, err := resource.newCreateChange(context.TODO(), tc.Obj, tc.Cur, tc.Des)
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
			Obj: &providerv2alpha1.AzureConfig{
				Spec: providerv2alpha1.AzureConfigSpec{
					Cluster: providerv2alpha1.Cluster{
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
			Obj: &providerv2alpha1.AzureConfig{
				Spec: providerv2alpha1.AzureConfigSpec{
					Cluster: providerv2alpha1.Cluster{
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

	var resource *Resource
	{
		c := Config{
			Logger: microloggertest.New(),

			Azure:            testAzure,
			AzureConfig:      fakeclient.NewAzureConfig(),
			InstallationName: "test-installation",
		}

		resource, err = New(c)
		if err != nil {
			t.Fatalf("expected '%#v' got '%#v'", nil, err)
		}
	}

	for i, tc := range testCases {
		resourceGroup, err := resource.newDeleteChange(context.TODO(), tc.Obj, tc.Cur, tc.Des)
		if err != nil {
			t.Fatalf("case %d expected '%v' got '%#v'", i+1, nil, err)
		}

		if !reflect.DeepEqual(resourceGroup, tc.ExpectedResourceGroup) {
			t.Fatalf("case %d expected %#v, got %#v", i+1, resourceGroup, tc.ExpectedResourceGroup)
		}
	}
}
