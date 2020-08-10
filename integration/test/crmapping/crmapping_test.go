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
	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/giantswarm/azure-operator/v4/service/controller/resource/capzcrs"
)

func TestAzureConfigCRMapping(t *testing.T) {
	testCases := []struct {
		name         string
		inputFiles   []string
		errorMatcher func(error) bool
	}{
		{
			name: "case 0: Simple AzureConfig",
			inputFiles: []string{
				"simple_azureconfig.yaml",
			},
			errorMatcher: nil,
		},
	}

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Log(tc.name)

			client := newFakeClient()
			ensureCRsExist(t, client, tc.inputFiles)

			r := constructResource(t, client)

			err := r.EnsureCreated(context.Background(), nil)

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

	m := make(map[string]interface{})
	err = yaml.Unmarshal(bs, m)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	switch m["kind"] {
	case "AzureConfig":
		obj = &providerv1alpha1.AzureConfig{}
	default:
		return nil, microerror.Maskf(unknownKindError, "kind: %s", m["kind"])
	}

	err = yaml.Unmarshal(bs, obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return obj, nil
}

func newFakeClient() client.Client {
	scheme := runtime.NewScheme()
	providerv1alpha1.AddToScheme(scheme)

	return fake.NewFakeClientWithScheme(scheme)
}
