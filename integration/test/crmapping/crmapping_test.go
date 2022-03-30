package crmapping

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"testing"
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

	"github.com/giantswarm/azure-operator/v5/service/controller/azurecluster/handler/azureconfig"
	"github.com/giantswarm/azure-operator/v5/service/controller/azureconfig/handler/capzcrs"
)

var update = flag.Bool("update", false, "update .golden reference files")

// Test_BidirectionalAzureConfigCRMapping uses golden files.
//
//  go test ./integration/test/crmapping -run Test_BidirectionalAzureConfigCRMapping -update
//
func Test_BidirectionalAzureConfigCRMapping(t *testing.T) {
	testCases := []struct {
		name              string
		location          string
		clusterID         string
		namespace         string
		preConditionFiles []string
	}{
		{
			name:      "case 0: migrate AzureConfig",
			location:  "westeurope",
			clusterID: "c6fme",
			namespace: "org-giantswarm",
			preConditionFiles: []string{
				"namespaces.yaml",
				"simple_azureconfig.yaml",
			},
		},
	}

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Log(tc.name)

			client := newFakeClient()
			ensureCRsExist(t, client, tc.preConditionFiles)

			// Reconcile AzureConfig.
			{
				obj := &providerv1alpha1.AzureConfig{}
				err := client.Get(context.Background(), types.NamespacedName{Name: tc.clusterID, Namespace: metav1.NamespaceDefault}, obj)
				if err != nil {
					t.Fatal(microerror.JSON(err))
				}

				r := constructCapzcrsHandler(t, client, tc.location)

				err = r.EnsureCreated(context.Background(), obj)
				if err != nil {
					t.Fatal(microerror.JSON(err))
				}
			}

			// Reconcile AzureCluster.
			{
				obj := &capzv1alpha3.AzureCluster{}
				err := client.Get(context.Background(), types.NamespacedName{Name: tc.clusterID, Namespace: tc.namespace}, obj)
				if err != nil {
					t.Fatal(microerror.JSON(err))
				}

				r := constructAzureConfigHandler(t, client)

				err = r.EnsureCreated(context.Background(), obj)
				if err != nil {
					t.Fatal(microerror.JSON(err))
				}
			}

			verifyCR(t, client, tc.name, new(providerv1alpha1.AzureConfig), types.NamespacedName{Name: tc.clusterID, Namespace: metav1.NamespaceDefault})
			verifyCR(t, client, tc.name, new(capiv1alpha3.Cluster), types.NamespacedName{Name: tc.clusterID, Namespace: tc.namespace})
			verifyCR(t, client, tc.name, new(capzv1alpha3.AzureCluster), types.NamespacedName{Name: tc.clusterID, Namespace: tc.namespace})
			verifyCR(t, client, tc.name, new(capzv1alpha3.AzureMachine), types.NamespacedName{Name: fmt.Sprintf("%s-master-0", tc.clusterID), Namespace: tc.namespace})
		})
	}
}

// Test_BidirectionalAzureClusterCRMapping uses golden files.
//
//  go test ./integration/test/crmapping -run Test_BidirectionalAzureClusterCRMapping -update
//
func Test_BidirectionalAzureClusterCRMapping(t *testing.T) {
	testCases := []struct {
		name              string
		location          string
		clusterID         string
		namespace         string
		preConditionFiles []string
	}{
		{
			name:      "case 0: create AzureConfig",
			location:  "westeurope",
			clusterID: "c6fme",
			namespace: "org-giantswarm",
			preConditionFiles: []string{
				"namespaces.yaml",
				"simple_azurecluster.yaml",
				"simple_azuremachine.yaml",
				"simple_cluster.yaml",
			},
		},
	}

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Log(tc.name)

			client := newFakeClient()
			ensureCRsExist(t, client, tc.preConditionFiles)

			// Reconcile AzureCluster.
			{
				obj := &capzv1alpha3.AzureCluster{}
				err := client.Get(context.Background(), types.NamespacedName{Name: tc.clusterID, Namespace: tc.namespace}, obj)
				if err != nil {
					t.Fatal(microerror.JSON(err))
				}

				r := constructAzureConfigHandler(t, client)

				err = r.EnsureCreated(context.Background(), obj)
				if err != nil {
					t.Fatal(microerror.JSON(err))
				}
			}

			// Reconcile AzureConfig.
			{
				obj := &providerv1alpha1.AzureConfig{}
				err := client.Get(context.Background(), types.NamespacedName{Name: tc.clusterID, Namespace: metav1.NamespaceDefault}, obj)
				if err != nil {
					t.Fatal(microerror.JSON(err))
				}

				r := constructCapzcrsHandler(t, client, tc.location)

				err = r.EnsureCreated(context.Background(), obj)
				if err != nil {
					t.Fatal(microerror.JSON(err))
				}
			}

			verifyCR(t, client, tc.name, new(providerv1alpha1.AzureConfig), types.NamespacedName{Name: tc.clusterID, Namespace: metav1.NamespaceDefault})
			verifyCR(t, client, tc.name, new(capiv1alpha3.Cluster), types.NamespacedName{Name: tc.clusterID, Namespace: tc.namespace})
			verifyCR(t, client, tc.name, new(capzv1alpha3.AzureCluster), types.NamespacedName{Name: tc.clusterID, Namespace: tc.namespace})
			verifyCR(t, client, tc.name, new(capzv1alpha3.AzureMachine), types.NamespacedName{Name: fmt.Sprintf("%s-master-0", tc.clusterID), Namespace: tc.namespace})
		})
	}
}

func constructCapzcrsHandler(t *testing.T, client client.Client, location string) resource.Interface {
	t.Helper()

	c := capzcrs.Config{
		CtrlClient: client,
		Logger:     microloggertest.New(),

		Location: location,
	}

	r, err := capzcrs.New(c)
	if err != nil {
		t.Fatal(err)
	}

	return r
}

func constructAzureConfigHandler(t *testing.T, client client.Client) resource.Interface {
	t.Helper()

	c := azureconfig.Config{
		CtrlClient: client,
		Logger:     microloggertest.New(),

		APIServerSecurePort: 443,
		Calico: azureconfig.CalicoConfig{
			CIDRSize: 16,
			MTU:      1430,
			Subnet:   "10.1.0.0/16",
		},
		ClusterIPRange:                 "172.31.0.0/16",
		EtcdPrefix:                     "giantswarm.io",
		ManagementClusterResourceGroup: "ghost",
		VnetMaskSize:                   16,
	}

	r, err := azureconfig.New(c)
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

		if o.GetObjectKind().GroupVersionKind().Kind == "NamespaceList" {
			lst := o.(*corev1.NamespaceList)
			for i := range lst.Items {
				err = client.Create(context.Background(), &lst.Items[i])
				if err != nil {
					t.Fatalf("failed to create object from input file %s: %#v", f, err)
				}
			}
			continue
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
	case "Namespace":
		obj = new(corev1.Namespace)
	case "NamespaceList":
		obj = new(corev1.NamespaceList)
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

	objMap := make(map[string]interface{})
	err = yaml.Unmarshal(out, objMap)
	if err != nil {
		t.Fatalf("err = %#q, want %#v", microerror.JSON(err), nil)
	}

	goldenMap := make(map[string]interface{})
	err = yaml.Unmarshal(goldenFile, goldenMap)
	if err != nil {
		t.Fatalf("err = %#q, want %#v", microerror.JSON(err), nil)
	}

	if !cmp.Equal(objMap, goldenMap) {
		t.Fatalf("\n\n%s\n", cmp.Diff(goldenMap, objMap))
	}
}
