package crmapping

import (
	"context"
	"io/ioutil"
	"path"
	"strconv"
	"testing"

	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger/microloggertest"
	"github.com/giantswarm/operatorkit/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	capzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/azure-operator/v4/service/controller/resource/capzcrs"
)

func TestAzureConfigCRMapping(t *testing.T) {
	testCases := []struct {
		name            string
		azureConfigFile string
		preTestFiles    []string
		errorMatcher    func(error) bool
	}{
		{
			name:            "case 0: Simple AzureConfig",
			azureConfigFile: "simple_azureconfig.yaml",
			preTestFiles:    []string{},
			errorMatcher:    nil,
		},
	}

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Log(tc.name)

			client := newFakeClient()
			ensureCRsExist(t, client, tc.preTestFiles)

			o, err := loadCR(tc.azureConfigFile)
			if err != nil {
				t.Fatal(err)
			}

			azureConfig, ok := o.(*providerv1alpha1.AzureConfig)
			if !ok {
				t.Fatalf("couldn't cast object %T to %T", o, azureConfig)
			}

			r := constructResource(t, client)

			err = r.EnsureCreated(context.Background(), azureConfig)

			// TODO: Read expected CRs w/ client and compare to "golden" versions.

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
		})
	}
}

func constructResource(t *testing.T, client client.Client) resource.Interface {
	t.Helper()

	c := capzcrs.Config{
		CtrlClient: client,
		Logger:     microloggertest.New(),
	}

	r, err := capzcrs.New(c)
	if err != nil {
		t.Fatal(err)
	}

	return r
}

func ensureCRsExist(t *testing.T, client client.Client, inputFiles []string) {
	for _, f := range inputFiles {
		o, err := loadCR(f)
		if err != nil {
			t.Fatalf("failed to load input file %s: %#v", f, err)
		}

		err = client.Create(context.Background(), o)
		if err != nil {
			t.Fatalf("failed to create object from input file %s: %#v", f, err)
		}
	}
}

func loadCR(fName string) (runtime.Object, error) {
	var err error
	var obj runtime.Object

	var bs []byte
	{
		bs, err = ioutil.ReadFile(path.Join("testdata", fName))
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	// First parse kind.
	t := &metav1.TypeMeta{}
	err = yaml.Unmarshal(bs, t)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// Then construct correct CR object.
	switch t.Kind {
	case "AzureConfig":
		obj = new(providerv1alpha1.AzureConfig)
	default:
		return nil, microerror.Maskf(unknownKindError, "kind: %s", t.Kind)
	}

	// ...and unmarshal the whole object.
	err = yaml.Unmarshal(bs, obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return obj, nil
}

func newFakeClient() client.Client {
	scheme := runtime.NewScheme()
	err := providerv1alpha1.AddToScheme(scheme)
	if err != nil {
		panic(err)
	}

	err = capiv1alpha3.AddToScheme(scheme)
	if err != nil {
		panic(err)
	}

	err = capzv1alpha3.AddToScheme(scheme)
	if err != nil {
		panic(err)
	}

	return fake.NewFakeClientWithScheme(scheme)
}
