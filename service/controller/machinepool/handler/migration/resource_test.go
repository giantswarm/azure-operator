package migration

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"testing"
	"time"
	"unicode"

	oldcapiexpv1alpha3 "github.com/giantswarm/apiextensions/v5/pkg/apis/capiexp/v1alpha3"
	"github.com/giantswarm/microerror"
	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	capiexp "sigs.k8s.io/cluster-api/exp/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

var update = flag.Bool("update", false, "update .golden reference files")

type expected struct {
	newMachinePoolModificationsCount int
	oldMachinePoolDeleted            bool
}

func loadMachinePoolCR(fName string) (*capiexp.MachinePool, error) {
	var err error
	var obj *capiexp.MachinePool

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

	switch t.Kind {
	case "MachinePool":
		obj = new(capiexp.MachinePool)
	default:
		return nil, microerror.Maskf(unknownKindError, "kind: %s", t.Kind)
	}

	// ...and unmarshal the whole object.
	err = yaml.Unmarshal(bs, &obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return obj, nil
}

func loadOldMachinePoolCR(fName string) (*oldcapiexpv1alpha3.MachinePool, error) {
	var err error
	var obj *oldcapiexpv1alpha3.MachinePool

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

	switch t.Kind {
	case "MachinePool":
		obj = new(oldcapiexpv1alpha3.MachinePool)
	default:
		return nil, microerror.Maskf(unknownKindError, "kind: %s", t.Kind)
	}

	// ...and unmarshal the whole object.
	err = yaml.Unmarshal(bs, &obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return obj, nil
}

func verifyCR(t *testing.T, client client.Client, testName string, obj client.Object, nsName types.NamespacedName) {
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
