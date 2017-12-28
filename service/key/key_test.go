package key

import (
	"fmt"
	"testing"

	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
)

func Test_AzureCloudType(t *testing.T) {
	customObject := providerv1alpha1.AzureConfig{
		Spec: providerv1alpha1.AzureConfigSpec{
			Cluster: providerv1alpha1.Cluster{},
		},
	}

	actualRes := AzureCloudType(customObject)
	if actualRes != defaultAzureCloudType {
		t.Fatalf("Expected cloud type %s but was %s", defaultAzureCloudType, actualRes)
	}
}

func Test_ClusterID(t *testing.T) {
	expectedID := "test-cluster"

	customObject := providerv1alpha1.AzureConfig{
		Spec: providerv1alpha1.AzureConfigSpec{
			Cluster: providerv1alpha1.Cluster{
				ID: expectedID,
				Customer: providerv1alpha1.ClusterCustomer{
					ID: "test-customer",
				},
			},
		},
	}

	if ClusterID(customObject) != expectedID {
		t.Fatalf("Expected cluster ID %s but was %s", expectedID, ClusterID(customObject))
	}
}

func Test_ClusterCustomer(t *testing.T) {
	expectedID := "test-customer"

	customObject := providerv1alpha1.AzureConfig{
		Spec: providerv1alpha1.AzureConfigSpec{
			Cluster: providerv1alpha1.Cluster{
				ID: "test-cluster",
				Customer: providerv1alpha1.ClusterCustomer{
					ID: expectedID,
				},
			},
		},
	}

	if ClusterCustomer(customObject) != expectedID {
		t.Fatalf("Expected customer ID %s but was %s", expectedID, ClusterCustomer(customObject))
	}
}

func Test_KeyVaultName(t *testing.T) {
	expectedVaultName := "test-cluster-vault"

	customObject := providerv1alpha1.AzureConfig{
		Spec: providerv1alpha1.AzureConfigSpec{
			Cluster: providerv1alpha1.Cluster{
				ID: "test-cluster",
			},
		},
	}

	if KeyVaultName(customObject) != expectedVaultName {
		t.Fatalf("Expected key vault name %s but was %s", expectedVaultName, KeyVaultName(customObject))
	}
}

func Test_KeyVaultKey(t *testing.T) {
	var (
		path        = "/etc/kubernetes/ssl/calico/client-ca.pem"
		expectedKey = "etc-kubernetes-ssl-calico-client-ca-pem"
	)

	key := KeyVaultKey(path)

	if key != expectedKey {
		t.Errorf("expected key = %q, got %q", expectedKey, key)
	}
}

func Test_Location(t *testing.T) {
	expectedLocation := "West Europe"

	customObject := providerv1alpha1.AzureConfig{
		Spec: providerv1alpha1.AzureConfigSpec{
			Azure: providerv1alpha1.AzureConfigSpecAzure{
				Location: expectedLocation,
			},
			Cluster: providerv1alpha1.Cluster{
				ID: "test-cluster",
			},
		},
	}

	if Location(customObject) != expectedLocation {
		t.Fatalf("Expected location %s but was %s", expectedLocation, Location(customObject))
	}
}

func Test_Functions_for_AzureResourceKeys(t *testing.T) {
	clusterID := "test-cluster"

	testCases := []struct {
		Func           func(providerv1alpha1.AzureConfig) string
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

	customObject := providerv1alpha1.AzureConfig{
		Spec: providerv1alpha1.AzureConfigSpec{
			Cluster: providerv1alpha1.Cluster{
				ID: "test-cluster",
			},
		},
	}

	for _, tc := range testCases {
		actualRes := tc.Func(customObject)
		if actualRes != tc.ExpectedResult {
			t.Fatalf("Expected %s but was %s", tc.ExpectedResult, actualRes)
		}
	}
}
