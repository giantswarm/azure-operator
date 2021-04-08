package capzcrs

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
	"time"
	"unicode"

	providerv1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger/microloggertest"
	"github.com/giantswarm/operatorkit/v4/pkg/resource"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	capzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake" // nolint:staticcheck
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

var update = flag.Bool("update", false, "update .golden reference files")

// Test_AzureConfigCRMapping uses golden files.
//
//  go test ./service/controller/azureconfig/handler/capzcrs -run Test_AzureConfigCRMapping -update
//
func Test_AzureConfigCRMapping(t *testing.T) {
	testCases := []struct {
		name            string
		location        string
		azureConfigFile string
		preTestFiles    []string
		errorMatcher    func(error) bool
	}{
		{
			name:            "case 0: Simple AzureConfig migration to create CAPI & CAPZ CRs",
			location:        "westeurope",
			azureConfigFile: "simple_azureconfig.yaml",
			preTestFiles:    []string{},
			errorMatcher:    nil,
		},
		{
			name:            "case 1: Simple AzureConfig reconciliation to verify existing CAPI & CAPZ CRs",
			location:        "westeurope",
			azureConfigFile: "simple_azureconfig.yaml",
			preTestFiles: []string{
				"simple_azurecluster.yaml",
				"simple_azuremachine.yaml",
				"simple_cluster.yaml",
			},
			errorMatcher: nil,
		},
		{
			name:            "case 2: Modified AzureConfig reconciliation to update existing CAPI & CAPZ CRs",
			location:        "westeurope",
			azureConfigFile: "modified_azureconfig.yaml",
			preTestFiles: []string{
				"simple_azurecluster.yaml",
				"simple_azuremachine.yaml",
				"simple_cluster.yaml",
			},
			errorMatcher: nil,
		},
		{
			name:            "case 3: Ensure that AzureCluster Subnets are not wiped",
			location:        "westeurope",
			azureConfigFile: "simple_azureconfig.yaml",
			preTestFiles: []string{
				"azurecluster_with_subnets.yaml",
				"simple_azuremachine.yaml",
				"simple_cluster.yaml",
			},
			errorMatcher: nil,
		},
		{

			name:            "case 4: Ensure that AzureCluster VNET gets updated",
			location:        "westeurope",
			azureConfigFile: "simple_azureconfig.yaml",
			preTestFiles: []string{
				"azurecluster_without_vnet.yaml",
				"simple_azuremachine.yaml",
				"simple_cluster.yaml",
			},
			errorMatcher: nil,
		},
		{
			name:            "case 5: Ensure that AzureCluster partial VNET gets updated",
			location:        "westeurope",
			azureConfigFile: "simple_azureconfig.yaml",
			preTestFiles: []string{
				"azurecluster_with_partial_vnet.yaml",
				"simple_azuremachine.yaml",
				"simple_cluster.yaml",
			},
			errorMatcher: nil,
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

			r := constructCapzcrsHandler(t, client, tc.location)

			err = r.EnsureCreated(context.Background(), azureConfig)
			if err != nil {
				t.Fatal(err)
			}

			verifyCR(t, client, tc.name, new(capiv1alpha3.Cluster), types.NamespacedName{Name: azureConfig.Name, Namespace: key.OrganizationNamespace(azureConfig)})
			verifyCR(t, client, tc.name, new(capzv1alpha3.AzureCluster), types.NamespacedName{Name: azureConfig.Name, Namespace: key.OrganizationNamespace(azureConfig)})
			verifyCR(t, client, tc.name, new(capzv1alpha3.AzureMachine), types.NamespacedName{Name: fmt.Sprintf("%s-master-0", azureConfig.Name), Namespace: key.OrganizationNamespace(azureConfig)})

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

func Test_mergeAzureCluster(t *testing.T) {
	testCases := []struct {
		name         string
		original     *capzv1alpha3.AzureCluster
		desired      *capzv1alpha3.AzureCluster
		expected     *capzv1alpha3.AzureCluster
		errorMatcher func(error) bool
	}{
		{
			name:         "case 0: empty values",
			original:     &capzv1alpha3.AzureCluster{},
			desired:      &capzv1alpha3.AzureCluster{},
			expected:     &capzv1alpha3.AzureCluster{},
			errorMatcher: nil,
		},
		{
			name: "case 1: preserve subnets",
			original: &capzv1alpha3.AzureCluster{
				Spec: capzv1alpha3.AzureClusterSpec{
					NetworkSpec: capzv1alpha3.NetworkSpec{
						Subnets: []*capzv1alpha3.SubnetSpec{
							&capzv1alpha3.SubnetSpec{
								ID:   "nodepool-np201",
								Role: capzv1alpha3.SubnetNode,
							},
						},
					},
				},
			},
			desired: &capzv1alpha3.AzureCluster{},
			expected: &capzv1alpha3.AzureCluster{
				Spec: capzv1alpha3.AzureClusterSpec{
					NetworkSpec: capzv1alpha3.NetworkSpec{
						Subnets: []*capzv1alpha3.SubnetSpec{
							&capzv1alpha3.SubnetSpec{
								ID:   "nodepool-np201",
								Role: capzv1alpha3.SubnetNode,
							},
						},
					},
				},
			},
			errorMatcher: nil,
		},
	}

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Log(tc.name)

			o, err := mergeAzureCluster(tc.original, tc.desired)

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

			if !cmp.Equal(o, tc.expected) {
				t.Fatalf("\n\n%s\n", cmp.Diff(tc.expected, o))
			}
		})
	}
}

func constructCapzcrsHandler(t *testing.T, client client.Client, location string) resource.Interface {
	t.Helper()

	c := Config{
		CtrlClient: client,
		Logger:     microloggertest.New(),

		Location: location,
	}

	r, err := New(c)
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
		bs, err = ioutil.ReadFile(filepath.Join("testdata", fName))
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
	case "Cluster":
		obj = new(capiv1alpha3.Cluster)
	case "AzureConfig":
		obj = new(providerv1alpha1.AzureConfig)
	case "AzureCluster":
		obj = new(capzv1alpha3.AzureCluster)
	case "AzureMachine":
		obj = new(capzv1alpha3.AzureMachine)
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

	err := capiv1alpha3.AddToScheme(scheme)
	if err != nil {
		panic(err)
	}

	err = capzv1alpha3.AddToScheme(scheme)
	if err != nil {
		panic(err)
	}

	err = corev1.AddToScheme(scheme)
	if err != nil {
		panic(err)
	}

	err = providerv1alpha1.AddToScheme(scheme)
	if err != nil {
		panic(err)
	}

	return fake.NewFakeClientWithScheme(scheme)
}

// normalizeFileName converts all non-digit, non-letter runes in input string to
// dash ('-'). Coalesces multiple dashes into one.
func normalizeFileName(s string) string {
	var result []rune
	for _, r := range s {
		if unicode.IsDigit(r) || unicode.IsLetter(r) {
			result = append(result, r)
		} else {
			l := len(result)
			if l > 0 && result[l-1] != '-' {
				result = append(result, rune('-'))
			}
		}
	}
	return string(result)
}

func verifyCR(t *testing.T, client client.Client, testName string, obj runtime.Object, nsName types.NamespacedName) {
	t.Helper()

	err := client.Get(context.Background(), nsName, obj)
	if err != nil {
		t.Fatalf("err = %#q, want %#v", microerror.JSON(err), nil)
	}

	ta, err := meta.TypeAccessor(obj)
	if err != nil {
		t.Fatalf("err = %#q, want %#v", microerror.JSON(err), nil)
	}

	out, err := yaml.Marshal(obj)
	if err != nil {
		t.Fatalf("err = %#q, want %#v", microerror.JSON(err), nil)
	}

	p := filepath.Join("testdata", fmt.Sprintf("%s_%s.golden", normalizeFileName(testName), ta.GetKind()))

	if *update {
		err = ioutil.WriteFile(p, out, 0644) // nolint:gosec
		if err != nil {
			t.Fatalf("err = %#q, want %#v", microerror.JSON(err), nil)
		}
	}

	goldenFile, err := ioutil.ReadFile(p)
	if err != nil {
		t.Fatalf("err = %#q, want %#v", microerror.JSON(err), nil)
	}

	//
	// XXX: Workaround to exclude timestamp differences from comparison.
	//		Currently e.g. AzureConfig.Status.Cluster has few fields with
	//		lastTransitionTime that contains timestamp. When executing e.g.
	//		test that creates AzureConfig from CAPI + CAPZ types, it injects
	//		time.Now() in those fields and this would always differ from golden
	//		files that represent one point in time.
	//
	re := regexp.MustCompile(`lastTransitionTime: "\d+-\d+-\d+T\d+:\d+:\d+Z"`)
	timeNow := time.Now()
	goldenFile = re.ReplaceAll(goldenFile, []byte(fmt.Sprintf(`lastTransitionTime: "%s"`, timeNow)))
	out = re.ReplaceAll(out, []byte(fmt.Sprintf(`lastTransitionTime: "%s"`, timeNow)))

	// Final comparison of golden version vs. one generated by test.
	if !bytes.Equal(out, goldenFile) {
		t.Fatalf("\n\n%s\n", cmp.Diff(string(goldenFile), string(out)))
	}
}
