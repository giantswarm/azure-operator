package deployment

import (
	"context"
	"reflect"
	"testing"

	providerv2alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/certs/certstest"
	"github.com/giantswarm/micrologger/microloggertest"
	"github.com/giantswarm/randomkeys/randomkeystest"

	"github.com/giantswarm/azure-operator/client/fakeclient"
	"github.com/giantswarm/azure-operator/service/controller/setting"
	"github.com/giantswarm/azure-operator/service/controller/v2/cloudconfig"
)

var testAzure = setting.Azure{
	HostCluster: setting.AzureHostCluster{
		CIDR:           "10.0.0.0/8",
		ResourceGroup:  "test-group",
		VirtualNetwork: "test-vnet",
	},
	Location: "westeurope",
}

func Test_Resource_Deployment_GetDesiredState(t *testing.T) {
	customObject := &providerv2alpha1.AzureConfig{
		Spec: providerv2alpha1.AzureConfigSpec{
			Cluster: providerv2alpha1.Cluster{
				ID: "5xchu",
			},
		},
	}
	expectedDeployments := []deployment{
		deployment{
			Name:          "cluster-main-template",
			ResourceGroup: "5xchu",
		},
	}

	var err error

	var resource *Resource
	{
		var cloudConfig *cloudconfig.CloudConfig
		{
			c := cloudconfig.Config{
				CertsSearcher:      certstest.NewSearcher(),
				Logger:             microloggertest.New(),
				RandomkeysSearcher: randomkeystest.NewSearcher(),

				Azure:       testAzure,
				AzureConfig: fakeclient.NewAzureConfig(),
			}

			cloudConfig, err = cloudconfig.New(c)
			if err != nil {
				t.Fatalf("expected '%#v' got '%#v'", nil, err)
			}
		}

		c := Config{
			CloudConfig: cloudConfig,
			Logger:      microloggertest.New(),

			Azure:           testAzure,
			AzureConfig:     fakeclient.NewAzureConfig(),
			TemplateVersion: "master",
		}

		resource, err = New(c)
		if err != nil {
			t.Fatalf("expected '%#v' got '%#v'", nil, err)
		}
	}

	result, err := resource.GetDesiredState(context.TODO(), customObject)
	if err != nil {
		t.Fatalf("expected '%v' got '%#v'", nil, err)
	}

	deployments, ok := result.([]deployment)
	if !ok {
		t.Fatalf("case expected '%T', got '%T'", []deployment{}, deployments)
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

func Test_Resource_Deployment_newCreateChange(t *testing.T) {
	testCases := []struct {
		Obj                 interface{}
		Cur                 interface{}
		Des                 interface{}
		ExpectedDeployments []deployment
	}{
		{
			// Case 1. Current state is empty. A deployment is created.
			Obj: &providerv2alpha1.AzureConfig{
				Spec: providerv2alpha1.AzureConfigSpec{
					Cluster: providerv2alpha1.Cluster{
						ID: "5xchu",
					},
				},
			},
			Cur: []deployment{},
			Des: []deployment{
				deployment{
					Name: "cluster-setup",
				},
			},
			ExpectedDeployments: []deployment{
				deployment{
					Name: "cluster-setup",
				},
			},
		},
		{
			// Case 2. Current and desired states are the same.
			// No deployments are created.
			Obj: &providerv2alpha1.AzureConfig{
				Spec: providerv2alpha1.AzureConfigSpec{
					Cluster: providerv2alpha1.Cluster{
						ID: "5xchu",
					},
				},
			},
			Cur: []deployment{
				deployment{
					Name: "cluster-setup",
				},
			},
			Des: []deployment{
				deployment{
					Name: "cluster-setup",
				},
			},
			ExpectedDeployments: nil,
		},
		{
			// Case 3. Current and desired states are different.
			// A new deployment is created.
			Obj: &providerv2alpha1.AzureConfig{
				Spec: providerv2alpha1.AzureConfigSpec{
					Cluster: providerv2alpha1.Cluster{
						ID: "5xchu",
					},
				},
			},
			Cur: []deployment{
				deployment{
					Name: "cluster-setup",
				},
			},
			Des: []deployment{
				deployment{
					Name: "cluster-setup",
				},
				deployment{
					Name: "network-setup",
				},
			},
			ExpectedDeployments: []deployment{
				deployment{
					Name: "network-setup",
				},
			},
		},
	}

	var err error

	var resource *Resource
	{
		var cloudConfig *cloudconfig.CloudConfig
		{
			c := cloudconfig.Config{
				CertsSearcher:      certstest.NewSearcher(),
				Logger:             microloggertest.New(),
				RandomkeysSearcher: randomkeystest.NewSearcher(),

				Azure:       testAzure,
				AzureConfig: fakeclient.NewAzureConfig(),
			}

			cloudConfig, err = cloudconfig.New(c)
			if err != nil {
				t.Fatalf("expected '%#v' got '%#v'", nil, err)
			}
		}

		c := Config{
			CloudConfig: cloudConfig,
			Logger:      microloggertest.New(),

			Azure:           testAzure,
			AzureConfig:     fakeclient.NewAzureConfig(),
			TemplateVersion: "master",
		}

		resource, err = New(c)
		if err != nil {
			t.Fatalf("expected '%#v' got '%#v'", nil, err)
		}
	}

	for i, tc := range testCases {
		deployments, err := resource.newCreateChange(context.TODO(), tc.Obj, tc.Cur, tc.Des)
		if err != nil {
			t.Fatalf("case %d expected %#v got %#v", i+1, nil, err)
		}

		//if !reflect.DeepEqual(deployments, tc.ExpectedDeployments) {
		if !reflect.DeepEqual(deployments, tc.ExpectedDeployments) {
			t.Fatalf("case %d expected %#v got %#v", i+1, tc.ExpectedDeployments, deployments)
		}
	}
}
