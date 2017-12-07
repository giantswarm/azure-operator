package key

import (
	"fmt"
	"testing"

	"github.com/giantswarm/certificatetpr"

	"github.com/giantswarm/azuretpr"
	azurespec "github.com/giantswarm/azuretpr/spec"
	"github.com/giantswarm/clustertpr"
	"github.com/giantswarm/clustertpr/spec"
)

func Test_AzureCloudType(t *testing.T) {
	customObject := azuretpr.CustomObject{
		Spec: azuretpr.Spec{
			Cluster: clustertpr.Spec{},
		},
	}

	actualRes := AzureCloudType(customObject)
	if actualRes != defaultAzureCloudType {
		t.Fatalf("Expected cloud type %s but was %s", defaultAzureCloudType, actualRes)
	}
}

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

func Test_KeyVaultName(t *testing.T) {
	expectedVaultName := "test-cluster-vault"

	customObject := azuretpr.CustomObject{
		Spec: azuretpr.Spec{
			Cluster: clustertpr.Spec{
				Cluster: spec.Cluster{
					ID: "test-cluster",
				},
			},
		},
	}

	if KeyVaultName(customObject) != expectedVaultName {
		t.Fatalf("Expected key vault name %s but was %s", expectedVaultName, KeyVaultName(customObject))
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

func Test_Functions_for_AzureResourceKeys(t *testing.T) {
	clusterID := "test-cluster"

	testCases := []struct {
		Func           func(azuretpr.CustomObject) string
		ExpectedResult string
	}{
		{
			Func:           MasterSecurityGroupName,
			ExpectedResult: fmt.Sprintf("%s-%s", clusterID, masterSecurityGroupSuffix),
		},
		{
			Func:           WorkerSecurityGroupName,
			ExpectedResult: fmt.Sprintf("%s-%s", clusterID, workerSecurityGroupSuffix),
		},
		{
			Func:           MasterSubnetName,
			ExpectedResult: fmt.Sprintf("%s-%s-%s", clusterID, virtualNetworkSuffix, masterSubnetSuffix),
		},
		{
			Func:           WorkerSubnetName,
			ExpectedResult: fmt.Sprintf("%s-%s-%s", clusterID, virtualNetworkSuffix, workerSubnetSuffix),
		},
		{
			Func:           RouteTableName,
			ExpectedResult: fmt.Sprintf("%s-%s", clusterID, routeTableSuffix),
		},
		{
			Func:           ResourceGroupName,
			ExpectedResult: clusterID,
		},
		{
			Func:           VnetName,
			ExpectedResult: fmt.Sprintf("%s-%s", clusterID, virtualNetworkSuffix),
		},
	}

	cluster := clustertpr.Spec{
		Cluster: spec.Cluster{
			ID: clusterID,
		},
	}

	customObject := azuretpr.CustomObject{
		Spec: azuretpr.Spec{
			Cluster: cluster,
		},
	}

	for _, tc := range testCases {
		actualRes := tc.Func(customObject)
		if actualRes != tc.ExpectedResult {
			t.Fatalf("Expected %s but was %s", tc.ExpectedResult, actualRes)
		}
	}
}

func Test_SecretName(t *testing.T) {
	expectedSecretName := "api-crt"

	assetKey := certificatetpr.AssetsBundleKey{
		Component: "api",
		Type:      "crt",
	}

	if SecretName(assetKey.Component, assetKey.Type) != expectedSecretName {
		t.Fatalf("Expected secret name %s but was %s", expectedSecretName, SecretName(assetKey.Component, assetKey.Type))
	}
}
