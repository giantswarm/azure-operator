package spark

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"unicode"

	"github.com/Azure/go-autorest/autorest/azure/auth"
	corev1alpha1 "github.com/giantswarm/apiextensions/v2/pkg/apis/core/v1alpha1"
	providerv1alpha1 "github.com/giantswarm/apiextensions/v2/pkg/apis/provider/v1alpha1"
	releasev1alpha1 "github.com/giantswarm/apiextensions/v2/pkg/apis/release/v1alpha1"
	"github.com/giantswarm/certs/v3/pkg/certstest"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger/microloggertest"
	"github.com/giantswarm/operatorkit/v2/pkg/resource"
	"github.com/giantswarm/randomkeys/v2/randomkeystest"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	capzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	capzexpv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/cluster-api/exp/api/v1alpha3"
	capiexpv1alpha3 "sigs.k8s.io/cluster-api/exp/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/azure-operator/v4/service/controller/resource/azureconfig"
	"github.com/giantswarm/azure-operator/v4/service/controller/setting"
)

var update = flag.Bool("update", false, "update .golden reference files")

// Test_AzureConfigCRMapping uses golden files.
//
//  go test ./service/controller/resource/capzcrs -run Test_AzureConfigCRMapping -update
//
func Test_CloudConfigRendering(t *testing.T) {
	testCases := []struct {
		name                 string
		azureMachinePoolFile string
		preTestFiles         []string
		errorMatcher         func(error) bool
	}{
		{
			name:                 "case 0: Spark",
			azureMachinePoolFile: "simple_azuremachinepool.yaml",
			preTestFiles: []string{
				"simple_cluster.yaml",
				"simple_azurecluster.yaml",
				"simple_machinepool.yaml",
				"simple_release.yaml",
				"encryption_key_secret.yaml",
				"simple_spark.yaml",
			},
			errorMatcher: nil,
		},
	}

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Log(tc.name)

			client := newFakeClient()
			ensureCRsExist(t, client, tc.preTestFiles)

			o, err := loadCR(tc.azureMachinePoolFile)
			if err != nil {
				t.Fatal(err)
			}

			azureMachinePool, ok := o.(*capzexpv1alpha3.AzureMachinePool)
			if !ok {
				t.Fatalf("couldn't cast object %T to %T", o, azureMachinePool)
			}

			r := constructSparkHandler(t, client)

			err = r.EnsureCreated(context.Background(), azureMachinePool)
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

			verifyCR(t, client, tc.name, new(corev1.Secret), types.NamespacedName{Name: secretName(azureMachinePool.Name), Namespace: azureMachinePool.Namespace})
		})
	}
}

func constructSparkHandler(t *testing.T, client client.Client) resource.Interface {
	t.Helper()

	randomKeysSearcher := randomkeystest.NewSearcher()

	c := Config{
		APIServerSecurePort: 443,
		Azure: setting.Azure{
			EnvironmentName: "AZUREPUBLICCLOUD",
			HostCluster: setting.AzureHostCluster{
				CIDR:                  "10.0.0.0/24",
				ResourceGroup:         "abcd",
				VirtualNetwork:        "abcd-VirtualNetwork",
				VirtualNetworkGateway: "abcd-NatGateway",
			},
			MSI:      setting.AzureMSI{},
			Location: "westeurope",
		},
		Calico: azureconfig.CalicoConfig{
			CIDRSize: 16,
			MTU:      1500,
			Subnet:   "",
		},
		CertsSearcher:      certstest.NewSearcher(certstest.Config{}),
		ClusterIPRange:     "10.0.0.0/16",
		EtcdPrefix:         "etcd",
		CredentialProvider: CredentialProviderStub{},
		CtrlClient:         client,
		Ignition: setting.Ignition{
			Path: fmt.Sprintf("%s/pkg/mod/github.com/giantswarm/k8scloudconfig/v7@v7.1.0", os.Getenv("GOPATH")),
		},
		Logger: microloggertest.New(),
		OIDC: setting.OIDC{
			ClientID:      "",
			IssuerURL:     "",
			UsernameClaim: "",
			GroupsClaim:   "",
		},
		RandomKeysSearcher: randomKeysSearcher,
		RegistryDomain:     "quay.io",
		SSHUserList:        "jose:publickey",
		SSOPublicKey:       "ssoPublicKey",
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
	case "AzureCluster":
		obj = new(capzv1alpha3.AzureCluster)
	case "AzureMachine":
		obj = new(capzv1alpha3.AzureMachine)
	case "MachinePool":
		obj = new(v1alpha3.MachinePool)
	case "AzureMachinePool":
		obj = new(capzexpv1alpha3.AzureMachinePool)
	case "Spark":
		obj = new(corev1alpha1.Spark)
	case "Release":
		obj = new(releasev1alpha1.Release)
	case "Secret":
		obj = new(corev1.Secret)
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

	err = capiexpv1alpha3.AddToScheme(scheme)
	if err != nil {
		panic(err)
	}

	err = capzexpv1alpha3.AddToScheme(scheme)
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

	err = corev1alpha1.AddToScheme(scheme)
	if err != nil {
		panic(err)
	}

	err = releasev1alpha1.AddToScheme(scheme)
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

	// Final comparison of golden version vs. one generated by test.
	if !bytes.Equal(out, goldenFile) {
		t.Fatalf("\n\n%s\n", cmp.Diff(string(goldenFile), string(out)))
	}
}

type CredentialProviderStub struct {
}

func (p CredentialProviderStub) GetOrganizationAzureCredentials(ctx context.Context, credentialNamespace, credentialName string) (auth.ClientCredentialsConfig, string, string, error) {
	return auth.ClientCredentialsConfig{
		ClientID:     "1234",
		ClientSecret: "abcd",
		TenantID:     "1234-abcd",
		AuxTenants:   nil,
		AADEndpoint:  "",
		Resource:     "",
	}, "1234-6789-1234-09876", "", nil
}
