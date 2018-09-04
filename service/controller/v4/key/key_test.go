package key

import (
	"fmt"
	"reflect"
	"strconv"
	"testing"

	"github.com/Azure/go-autorest/autorest/to"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
)

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

	expectedTags := map[string]*string{
		"GiantSwarmCluster":      to.StringPtr("test-cluster"),
		"GiantSwarmInstallation": to.StringPtr("test-install"),
		"GiantSwarmOrganization": to.StringPtr("test-org"),
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

func Test_Functions_for_DNSKeys(t *testing.T) {
	clusterID := "eggs2"
	baseDomainName := "domain.tld"
	resourceGroupName := clusterID

	testCases := []struct {
		Func           func(providerv1alpha1.AzureConfig) string
		ExpectedResult string
	}{
		{
			Func:           ClusterDNSDomain,
			ExpectedResult: fmt.Sprintf("%s.k8s.%s", clusterID, baseDomainName),
		},
		{
			Func:           DNSZoneAPI,
			ExpectedResult: baseDomainName,
		},
		{
			Func:           DNSZoneEtcd,
			ExpectedResult: baseDomainName,
		},
		{
			Func:           DNSZoneIngress,
			ExpectedResult: baseDomainName,
		},
		{
			Func:           DNSZonePrefixAPI,
			ExpectedResult: fmt.Sprintf("%s.k8s", clusterID),
		},
		{
			Func:           DNSZonePrefixEtcd,
			ExpectedResult: fmt.Sprintf("%s.k8s", clusterID),
		},
		{
			Func:           DNSZonePrefixIngress,
			ExpectedResult: fmt.Sprintf("%s.k8s", clusterID),
		},
		{
			Func:           DNSZoneResourceGroupAPI,
			ExpectedResult: resourceGroupName,
		},
		{
			Func:           DNSZoneResourceGroupEtcd,
			ExpectedResult: resourceGroupName,
		},
		{
			Func:           DNSZoneResourceGroupIngress,
			ExpectedResult: resourceGroupName,
		},
	}

	customObject := providerv1alpha1.AzureConfig{
		Spec: providerv1alpha1.AzureConfigSpec{
			Azure: providerv1alpha1.AzureConfigSpecAzure{
				DNSZones: providerv1alpha1.AzureConfigSpecAzureDNSZones{
					API: providerv1alpha1.AzureConfigSpecAzureDNSZonesDNSZone{
						ResourceGroup: resourceGroupName,
						Name:          baseDomainName,
					},
					Etcd: providerv1alpha1.AzureConfigSpecAzureDNSZonesDNSZone{
						ResourceGroup: resourceGroupName,
						Name:          baseDomainName,
					},
					Ingress: providerv1alpha1.AzureConfigSpecAzureDNSZonesDNSZone{
						ResourceGroup: resourceGroupName,
						Name:          baseDomainName,
					},
				},
			},
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

func Test_MasterNICName(t *testing.T) {
	expectedMasterNICName := "3p5j2-Master-1-NIC"

	customObject := providerv1alpha1.AzureConfig{
		Spec: providerv1alpha1.AzureConfigSpec{
			Cluster: providerv1alpha1.Cluster{
				ID: "3p5j2",
			},
		},
	}

	if MasterNICName(customObject) != expectedMasterNICName {
		t.Fatalf("Expected master nic name %s but was %s", expectedMasterNICName, MasterNICName(customObject))
	}
}

func Test_WorkerDockerVolumeSizeGB(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		customObject providerv1alpha1.AzureConfig
		expectedSize int
		errorMatcher func(error) bool
	}{
		{
			name: "case 0: worker with 350GB docker volume",
			customObject: providerv1alpha1.AzureConfig{
				Spec: providerv1alpha1.AzureConfigSpec{
					Azure: providerv1alpha1.AzureConfigSpecAzure{
						Workers: []providerv1alpha1.AzureConfigSpecAzureNode{
							{
								DockerVolumeSizeGB: "350",
							},
						},
					},
				},
			},
			expectedSize: 350,
			errorMatcher: nil,
		},
		{
			name: "case 1: no workers",
			customObject: providerv1alpha1.AzureConfig{
				Spec: providerv1alpha1.AzureConfigSpec{
					Azure: providerv1alpha1.AzureConfigSpecAzure{
						Workers: []providerv1alpha1.AzureConfigSpecAzureNode{},
					},
				},
			},
			expectedSize: 0,
			errorMatcher: nil,
		},
		{
			name: "case 2: invalid number",
			customObject: providerv1alpha1.AzureConfig{
				Spec: providerv1alpha1.AzureConfigSpec{
					Azure: providerv1alpha1.AzureConfigSpecAzure{
						Workers: []providerv1alpha1.AzureConfigSpecAzureNode{
							{
								DockerVolumeSizeGB: "foobar",
							},
						},
					},
				},
			},
			expectedSize: 0,
			errorMatcher: func(err error) bool {
				_, ok := microerror.Cause(err).(*strconv.NumError)
				return ok
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sz, err := WorkerDockerVolumeSizeGB(tc.customObject)

			switch {
			case err == nil && tc.errorMatcher == nil:
				// correct; carry on
			case err != nil && tc.errorMatcher == nil:
				t.Fatalf("error == %#v, want nil", err)
			case err == nil && tc.errorMatcher != nil:
				t.Fatalf("error == nil, want non-nil")
			case !tc.errorMatcher(err):
				t.Fatalf("error == %#v, want matching", err)
			}

			if sz != tc.expectedSize {
				t.Fatalf("Expected worker docker volume size  %d but was %d", tc.expectedSize, sz)
			}
		})
	}
}
