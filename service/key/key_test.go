package key

import (
	"testing"

	"github.com/giantswarm/azuretpr"
	azurespec "github.com/giantswarm/azuretpr/spec"
	"github.com/giantswarm/clustertpr"
	"github.com/giantswarm/clustertpr/spec"
)

func Test_ClusterID(t *testing.T) {
	expectedID := "test-cluster"

	cluster := clustertpr.Spec{
		Cluster: spec.Cluster{
			ID: expectedID,
		},
		Customer: spec.Customer{
			ID: "test-customer",
		},
	}

	customObject := azuretpr.CustomObject{
		Spec: azuretpr.Spec{
			Cluster: cluster,
		},
	}

	if ClusterID(customObject) != expectedID {
		t.Fatalf("Expected cluster ID %s but was %s", expectedID, ClusterID(customObject))
	}
}

func Test_ClusterCustomer(t *testing.T) {
	expectedID := "test-customer"

	cluster := clustertpr.Spec{
		Cluster: spec.Cluster{
			ID: "test-cluster",
		},
		Customer: spec.Customer{
			ID: expectedID,
		},
	}

	customObject := azuretpr.CustomObject{
		Spec: azuretpr.Spec{
			Cluster: cluster,
		},
	}

	if ClusterCustomer(customObject) != expectedID {
		t.Fatalf("Expected customer ID %s but was %s", expectedID, ClusterCustomer(customObject))
	}
}

func Test_Location(t *testing.T) {
	expectedLocation := "West Europe"

	cluster := clustertpr.Spec{
		Cluster: spec.Cluster{
			ID: "test-cluster",
		},
	}

	customObject := azuretpr.CustomObject{
		Spec: azuretpr.Spec{
			Azure: azurespec.Azure{
				Location: expectedLocation,
			},
			Cluster: cluster,
		},
	}

	if Location(customObject) != expectedLocation {
		t.Fatalf("Expected location %s but was %s", expectedLocation, Location(customObject))
	}
}
