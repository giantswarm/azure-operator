package key

import (
	"fmt"
	"reflect"
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

func Test_ClusterNamespace(t *testing.T) {
	expectedNamespace := "9dj1k"

	customObject := providerv1alpha1.AzureConfig{
		Spec: providerv1alpha1.AzureConfigSpec{
			Cluster: providerv1alpha1.Cluster{
				ID: expectedNamespace,
			},
		},
	}

	if ClusterNamespace(customObject) != expectedNamespace {
		t.Fatalf("Expected cluster namespace %s but was %s", expectedNamespace, ClusterNamespace(customObject))
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

func Test_ClusterOrganization(t *testing.T) {
	expectedID := "test-org"

	customObject := providerv1alpha1.AzureConfig{
		Spec: providerv1alpha1.AzureConfigSpec{
			Cluster: providerv1alpha1.Cluster{
				ID: "test-org",
				// Organization uses Customer until its renamed in the CRD.
				Customer: providerv1alpha1.ClusterCustomer{
					ID: expectedID,
				},
			},
		},
	}

	if ClusterOrganization(customObject) != expectedID {
		t.Fatalf("Expected organization %s but was %s", expectedID, ClusterOrganization(customObject))
	}
}

func Test_ClusterTags(t *testing.T) {
	installName := "test-install"

	expectedTags := map[string]string{
		"GiantSwarmCluster":      "test-cluster",
		"GiantSwarmInstallation": "test-install",
		"GiantSwarmOrganization": "test-org",
	}

	customObject := providerv1alpha1.AzureConfig{
		Spec: providerv1alpha1.AzureConfigSpec{
			Cluster: providerv1alpha1.Cluster{
				ID: "test-cluster",
				// Organization uses Customer until its renamed in the CRD.
				Customer: providerv1alpha1.ClusterCustomer{
					ID: "test-org",
				},
			},
		},
	}

	if !reflect.DeepEqual(expectedTags, ClusterTags(customObject, installName)) {
		t.Fatalf("Expected cluster tags %v but was %v", expectedTags, ClusterTags(customObject, installName))
	}
}

func Test_Functions_for_AzureResourceKeys(t *testing.T) {
	clusterID := "eggs2"

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
				ID: clusterID,
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
