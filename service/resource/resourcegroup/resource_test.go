package resourcegroup

import (
	"testing"

	"github.com/giantswarm/azuretpr"
	azuretprspec "github.com/giantswarm/azuretpr/spec"
	"github.com/giantswarm/clustertpr"
	"github.com/giantswarm/clustertpr/spec"
	"github.com/giantswarm/micrologger/microloggertest"

	"github.com/giantswarm/azure-operator/client"
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
			Obj: &azuretpr.CustomObject{
				Spec: azuretpr.Spec{
					Azure: azuretprspec.Azure{
						Location: "West Europe",
					},
					Cluster: clustertpr.Spec{
						Cluster: spec.Cluster{
							ID: "5xchu",
						},
						Customer: spec.Customer{
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
			Obj: &azuretpr.CustomObject{
				Spec: azuretpr.Spec{
					Azure: azuretprspec.Azure{
						Location: "East Asia",
					},
					Cluster: clustertpr.Spec{
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
		resourceConfig.AzureConfig = client.DefaultAzureConfig()
		resourceConfig.Logger = microloggertest.New()
		newResource, err = New(resourceConfig)
		if err != nil {
			t.Fatalf("expected '%#v' got '%#v'", nil, err)
		}

		for i, tc := range testCases {
			result, err := newResource.GetDesiredState(tc.Obj)
			if err != nil {
				t.Fatalf("case %d expected '%v' got '%#v'", i+1, nil, err)
			}

			group, ok := result.(*Group)
			if !ok {
				t.Fatalf("case %d expected '%T', got '%T'", i+1, &Group{}, group)
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
func Test_Resource_ResourceGroup_GetCreateState(t *testing.T) {
	testCases := []struct {
		Obj           interface{}
		Cur           interface{}
		Des           interface{}
		ExpectedGroup *Group
	}{
		{
			// Case 1. Current and desired states are the same. The resource
			// group should not be created.
			Obj: &azuretpr.CustomObject{
				Spec: azuretpr.Spec{
					Cluster: clustertpr.Spec{
						Cluster: spec.Cluster{
							ID: "5xchu",
						},
					},
				},
			},
			Cur: &Group{
				Name: "5xchu",
			},
			Des: &Group{
				Name: "5xchu",
			},
			ExpectedGroup: nil,
		},
		{
			// Case 2. Current state is nil. The resource group should be
			// created.
			Obj: &azuretpr.CustomObject{
				Spec: azuretpr.Spec{
					Cluster: clustertpr.Spec{
						Cluster: spec.Cluster{
							ID: "5xchu",
						},
					},
				},
			},
			Cur: &Group{},
			Des: &Group{
				Name: "5xchu",
			},
			ExpectedGroup: &Group{
				Name: "5xchu",
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
		result, err := newResource.GetCreateState(tc.Obj, tc.Cur, tc.Des)
		if err != nil {
			t.Fatalf("case %d expected '%v' got '%#v'", i+1, nil, err)
		}

		if tc.ExpectedGroup == nil {
			if tc.ExpectedGroup != result {
				t.Fatalf("case %d expected '%#v' got '%#v'", i+1, tc.ExpectedGroup, result)
			}
		} else {
			group, ok := result.(*Group)
			if !ok {
				t.Fatalf("case %d expected '%T', got '%T'", i+1, &Group{}, group)
			}
			if tc.ExpectedGroup.Name != group.Name {
				t.Fatalf("case %d expected '%s' got '%s'", i+1, tc.ExpectedGroup.Name, group.Name)
			}
		}
	}
}

func Test_Resource_ResourceGroup_GetDeleteState(t *testing.T) {
	testCases := []struct {
		Obj           interface{}
		Cur           interface{}
		Des           interface{}
		ExpectedGroup *Group
	}{
		{
			// Case 1. Current and desired states are the same. The resource
			// group should be deleted.
			Obj: &azuretpr.CustomObject{
				Spec: azuretpr.Spec{
					Cluster: clustertpr.Spec{
						Cluster: spec.Cluster{
							ID: "5xchu",
						},
					},
				},
			},
			Cur: &Group{
				Name: "5xchu",
			},
			Des: &Group{
				Name: "5xchu",
			},
			ExpectedGroup: &Group{
				Name: "5xchu",
			},
		},
		{
			// Case 2. Current state is nil. The resource group should not be
			// deleted.
			Obj: &azuretpr.CustomObject{
				Spec: azuretpr.Spec{
					Cluster: clustertpr.Spec{
						Cluster: spec.Cluster{
							ID: "5xchu",
						},
					},
				},
			},
			Cur: nil,
			Des: &Group{
				Name: "5xchu",
			},
			ExpectedGroup: nil,
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
		result, err := newResource.GetDeleteState(tc.Obj, tc.Cur, tc.Des)
		if err != nil {
			t.Fatalf("case %d expected '%v' got '%#v'", i+1, nil, err)
		}
		if tc.ExpectedGroup == nil {
			if tc.ExpectedGroup != result {
				t.Fatalf("case %d expected '%#v' got '%#v'", i+1, tc.ExpectedGroup, result)
			}
		} else {
			group, ok := result.(*Group)
			if !ok {
				t.Fatalf("case %d expected '%T', got '%T'", i+1, &Group{}, group)
			}
			if tc.ExpectedGroup.Name != group.Name {
				t.Fatalf("case %d expected '%s' got '%s'", i+1, tc.ExpectedGroup.Name, group.Name)
			}
		}
	}
}

func Test_Resource_ResourceGroup_GetUpdateState(t *testing.T) {
	testCases := []struct {
		Obj interface{}
		Cur interface{}
		Des interface{}
	}{
		{
			// Case 1. Current and desired states are the same. No changes
			// are returned as resource groups are not updated.
			Obj: &azuretpr.CustomObject{
				Spec: azuretpr.Spec{
					Cluster: clustertpr.Spec{
						Cluster: spec.Cluster{
							ID: "5xchu",
						},
					},
				},
			},
			Cur: &Group{
				Name: "5xchu",
			},
			Des: &Group{
				Name: "5xchu",
			},
		},
		{
			// Case 2. Current state is nil. No changes are returned as
			// resource groups are not updated.
			Obj: &azuretpr.CustomObject{
				Spec: azuretpr.Spec{
					Cluster: clustertpr.Spec{
						Cluster: spec.Cluster{
							ID: "5xchu",
						},
					},
				},
			},
			Cur: &Group{
				Name: "5xchu",
			},
			Des: &Group{
				Name: "5xchu",
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
		createState, deleteState, updateState, err := newResource.GetUpdateState(tc.Obj, tc.Cur, tc.Des)
		if err != nil {
			t.Fatalf("case %d expected '%v' got '%#v'", i+1, nil, err)
		}

		createGroup, ok := createState.(*Group)
		if !ok {
			t.Fatalf("case %d expected '%T', got '%T'", i+1, &Group{}, createGroup)
		}
		if createGroup.Name != "" {
			t.Fatalf("case %d expected create state '%#v' got '%#v'", i+1, &Group{}, createState)
		}

		deleteGroup, ok := deleteState.(*Group)
		if !ok {
			t.Fatalf("case %d expected '%T', got '%T'", i+1, &Group{}, deleteGroup)
		}
		if deleteGroup.Name != "" {
			t.Fatalf("case %d expected delete state '%#v' got '%#v'", i+1, &Group{}, deleteState)
		}

		updateGroup, ok := updateState.(*Group)
		if !ok {
			t.Fatalf("case %d expected '%T', got '%T'", i+1, &Group{}, updateGroup)
		}
		if updateGroup.Name != "" {
			t.Fatalf("case %d expected update state '%#v' got '%#v'", i+1, &Group{}, updateState)
		}
	}
}
