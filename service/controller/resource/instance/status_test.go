package instance

import (
	"reflect"
	"testing"

	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
)

func Test_Resource_Instance_computeForDeleteResourceStatus(t *testing.T) {
	testCases := []struct {
		name                   string
		customResource         providerv1alpha1.AzureConfig
		n                      string
		t                      string
		s                      string
		expectedCustomResource providerv1alpha1.AzureConfig
	}{
		{
			name: "case 0: Stage/InstancesUpgrading is removed when Stage/InstancesUpgrading should be removed",
			customResource: providerv1alpha1.AzureConfig{
				Status: providerv1alpha1.AzureConfigStatus{
					Cluster: providerv1alpha1.StatusCluster{
						Resources: []providerv1alpha1.StatusClusterResource{
							{
								Conditions: []providerv1alpha1.StatusClusterResourceCondition{
									{
										Type:   "Stage",
										Status: "InstancesUpgrading",
									},
								},
								Name: "instancev3",
							},
						},
					},
				},
			},
			n: "instancev3",
			t: "Stage",
			s: "InstancesUpgrading",
			expectedCustomResource: providerv1alpha1.AzureConfig{
				Status: providerv1alpha1.AzureConfigStatus{
					Cluster: providerv1alpha1.StatusCluster{
						Resources: nil,
					},
				},
			},
		},
		{
			name: "case 1: Stage/DeploymentInitialized is not removed when Stage/InstancesUpgrading should be removed",
			customResource: providerv1alpha1.AzureConfig{
				Status: providerv1alpha1.AzureConfigStatus{
					Cluster: providerv1alpha1.StatusCluster{
						Resources: []providerv1alpha1.StatusClusterResource{
							{
								Conditions: []providerv1alpha1.StatusClusterResourceCondition{
									{
										Type:   "Stage",
										Status: "DeploymentInitialized",
									},
								},
								Name: "instancev3",
							},
						},
					},
				},
			},
			n: "instancev3",
			t: "Stage",
			s: "InstancesUpgrading",
			expectedCustomResource: providerv1alpha1.AzureConfig{
				Status: providerv1alpha1.AzureConfigStatus{
					Cluster: providerv1alpha1.StatusCluster{
						Resources: []providerv1alpha1.StatusClusterResource{
							{
								Conditions: []providerv1alpha1.StatusClusterResourceCondition{
									{
										Type:   "Stage",
										Status: "DeploymentInitialized",
									},
								},
								Name: "instancev3",
							},
						},
					},
				},
			},
		},
		{
			name: "case 2: Stage/InstancesUpgrading is removed when Stage/InstancesUpgrading should be removed while Stage/DeploymentInitialized stays",
			customResource: providerv1alpha1.AzureConfig{
				Status: providerv1alpha1.AzureConfigStatus{
					Cluster: providerv1alpha1.StatusCluster{
						Resources: []providerv1alpha1.StatusClusterResource{
							{
								Conditions: []providerv1alpha1.StatusClusterResourceCondition{
									{
										Type:   "Stage",
										Status: "DeploymentInitialized",
									},
									{
										Type:   "Stage",
										Status: "InstancesUpgrading",
									},
								},
								Name: "instancev3",
							},
						},
					},
				},
			},
			n: "instancev3",
			t: "Stage",
			s: "InstancesUpgrading",
			expectedCustomResource: providerv1alpha1.AzureConfig{
				Status: providerv1alpha1.AzureConfigStatus{
					Cluster: providerv1alpha1.StatusCluster{
						Resources: []providerv1alpha1.StatusClusterResource{
							{
								Conditions: []providerv1alpha1.StatusClusterResourceCondition{
									{
										Type:   "Stage",
										Status: "DeploymentInitialized",
									},
								},
								Name: "instancev3",
							},
						},
					},
				},
			},
		},
		{
			name: "case 3: Stage/DeploymentInitialized of all resource versions is removed when Stage/DeploymentInitialized should be removed while Stage/InstancesUpgrading stays",
			customResource: providerv1alpha1.AzureConfig{
				Status: providerv1alpha1.AzureConfigStatus{
					Cluster: providerv1alpha1.StatusCluster{
						Resources: []providerv1alpha1.StatusClusterResource{
							{
								Conditions: []providerv1alpha1.StatusClusterResourceCondition{
									{
										Type:   "Stage",
										Status: "DeploymentInitialized",
									},
								},
								Name: "instancev3",
							},
							{
								Conditions: []providerv1alpha1.StatusClusterResourceCondition{
									{
										Type:   "Stage",
										Status: "DeploymentInitialized",
									},
									{
										Type:   "Stage",
										Status: "InstancesUpgrading",
									},
								},
								Name: "instancev4",
							},
						},
					},
				},
			},
			n: "instancev4",
			t: "Stage",
			s: "DeploymentInitialized",
			expectedCustomResource: providerv1alpha1.AzureConfig{
				Status: providerv1alpha1.AzureConfigStatus{
					Cluster: providerv1alpha1.StatusCluster{
						Resources: []providerv1alpha1.StatusClusterResource{
							{
								Conditions: []providerv1alpha1.StatusClusterResourceCondition{
									{
										Type:   "Stage",
										Status: "InstancesUpgrading",
									},
								},
								Name: "instancev4",
							},
						},
					},
				},
			},
		},
		{
			name: "case 4: same as 3 but also ensuring the preservation of resource statuses of other resources",
			customResource: providerv1alpha1.AzureConfig{
				Status: providerv1alpha1.AzureConfigStatus{
					Cluster: providerv1alpha1.StatusCluster{
						Resources: []providerv1alpha1.StatusClusterResource{
							{
								Conditions: []providerv1alpha1.StatusClusterResourceCondition{
									{
										Type:   "Stage",
										Status: "DeploymentInitialized",
									},
								},
								Name: "deploymentv3",
							},
							{
								Conditions: []providerv1alpha1.StatusClusterResourceCondition{
									{
										Type:   "Stage",
										Status: "DeploymentInitialized",
									},
								},
								Name: "deploymentv4",
							},
							{
								Conditions: []providerv1alpha1.StatusClusterResourceCondition{
									{
										Type:   "Stage",
										Status: "DeploymentInitialized",
									},
								},
								Name: "instancev3",
							},
							{
								Conditions: []providerv1alpha1.StatusClusterResourceCondition{
									{
										Type:   "Stage",
										Status: "DeploymentInitialized",
									},
									{
										Type:   "Stage",
										Status: "InstancesUpgrading",
									},
								},
								Name: "instancev4",
							},
						},
					},
				},
			},
			n: "instancev4",
			t: "Stage",
			s: "DeploymentInitialized",
			expectedCustomResource: providerv1alpha1.AzureConfig{
				Status: providerv1alpha1.AzureConfigStatus{
					Cluster: providerv1alpha1.StatusCluster{
						Resources: []providerv1alpha1.StatusClusterResource{
							{
								Conditions: []providerv1alpha1.StatusClusterResourceCondition{
									{
										Type:   "Stage",
										Status: "DeploymentInitialized",
									},
								},
								Name: "deploymentv3",
							},
							{
								Conditions: []providerv1alpha1.StatusClusterResourceCondition{
									{
										Type:   "Stage",
										Status: "DeploymentInitialized",
									},
								},
								Name: "deploymentv4",
							},
							{
								Conditions: []providerv1alpha1.StatusClusterResourceCondition{
									{
										Type:   "Stage",
										Status: "InstancesUpgrading",
									},
								},
								Name: "instancev4",
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			customResource := computeForDeleteResourceStatus(tc.customResource, tc.n, tc.t, tc.s)

			if !reflect.DeepEqual(customResource, tc.expectedCustomResource) {
				t.Fatalf("customResource == %#v, want %#v", customResource, tc.expectedCustomResource)
			}
		})
	}
}

func Test_Resource_Instance_unversionedName(t *testing.T) {
	e := "instance"
	n := unversionedName(Name)
	if n != e {
		t.Fatalf("unversionedName(Name) == %#q, want %#q", n, e)
	}
}
