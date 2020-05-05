package crd

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/giantswarm/microerror"
	"github.com/markbates/pkger"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
)

const (
	crdDirectory = "/config/crd/bases"
	crdKind      = "CustomResourceDefinition"
)

var (
	// GroupVersionKind of CustomResourceDefinition in apiextensions.k8s.io/v1beta1.
	v1beta1GroupVersionKind = schema.GroupVersionKind{
		Group:   apiextensions.GroupName,
		Version: "v1beta1",
		Kind:    crdKind,
	}
	// GroupVersionKind of CustomResourceDefinition in apiextensions.k8s.io/v1.
	v1GroupVersionKind = schema.GroupVersionKind{
		Group:   apiextensions.GroupName,
		Version: "v1",
		Kind:    crdKind,
	}
)

func Find(group, kind string) (interface{}, error) {
	// If a matching CRD is found during the walk, it will be saved to found.
	// This could be a v1 or v1beta1 CRD so it needs to be an interface{}.
	var found interface{}
	// Function called for every file in the CRD directory.
	walkFunc := func(fullPath string, info os.FileInfo, err error) error {
		// An unknown error, stop walking.
		if err != nil {
			return microerror.Mask(err)
		}
		// Skip directories and any other files after a match has been found.
		if found != nil || info.IsDir() {
			return nil
		}

		// pkger files have a path like github.com/giantswarm/apiextensions:/config/crd/bases/release.giantswarm.io_releases.yaml.
		split := strings.Split(fullPath, ":")
		path := split[1]
		extension := filepath.Ext(path)
		// Skip non-yaml files.
		if extension != ".yaml" {
			return nil
		}

		// Read the file to a string.
		yamlFile, err := pkger.Open(path)
		if err != nil {
			return microerror.Mask(err)
		}
		yamlString, err := ioutil.ReadAll(yamlFile)
		if err != nil {
			return microerror.Mask(err)
		}

		// Unmarshal into Unstructured since we don't know if this is a v1 or v1beta1 CRD yet.
		var object unstructured.Unstructured
		err = yaml.UnmarshalStrict(yamlString, &object)
		if err != nil {
			return microerror.Mask(err)
		}

		switch object.GetObjectKind().GroupVersionKind() {
		case v1beta1GroupVersionKind:
			var crd v1beta1.CustomResourceDefinition
			err = yaml.UnmarshalStrict(yamlString, &crd)
			if err != nil {
				return microerror.Mask(err)
			}
			if group == crd.Spec.Group && kind == crd.Spec.Names.Kind {
				found = &crd // Match, save results in outer scope
			}
			return nil
		case v1GroupVersionKind:
			var crd v1.CustomResourceDefinition
			err = yaml.UnmarshalStrict(yamlString, &crd)
			if err != nil {
				return microerror.Mask(err)
			}
			if group == crd.Spec.Group && kind == crd.Spec.Names.Kind {
				found = &crd // Match, save results in outer scope
			}
			return nil
		}
		return nil
	}

	// Entry point for walking the CRD YAML directory.
	err := pkger.Walk(crdDirectory, walkFunc)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	if found == nil {
		return nil, microerror.Mask(notFoundError)
	}

	return found, nil
}

// LoadV1Beta1 loads a v1beta1 CRD from the filesystem.
func LoadV1Beta1(group, kind string) *v1beta1.CustomResourceDefinition {
	found, err := Find(group, kind)
	if err != nil {
		panic(microerror.Mask(err))
	}
	crd, ok := found.(*v1beta1.CustomResourceDefinition)
	if !ok {
		panic(microerror.Mask(conversionFailedError))
	}
	return crd
}

// LoadV1Beta1 loads a v1 CRD from the filesystem
func LoadV1(group, kind string) (out *v1.CustomResourceDefinition) {
	found, err := Find(group, kind)
	if err != nil {
		panic(microerror.Mask(err))
	}
	crd, ok := found.(*v1.CustomResourceDefinition)
	if !ok {
		panic(microerror.Mask(conversionFailedError))
	}
	return crd
}
