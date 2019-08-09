package v8patch1

import (
	"github.com/giantswarm/versionbundle"
)

func VersionBundle() versionbundle.Bundle {
	return versionbundle.Bundle{
		Changelogs: []versionbundle.Changelog{
			{
				Component:   "kubernetes",
				Description: "Update kubernetes to 1.14.5 to fix CVE-2019-11247. https://github.com/kubernetes/kubernetes/blob/master/CHANGELOG-1.14.md",
				Kind:        versionbundle.KindChanged,
			},
		},
		Components: []versionbundle.Component{
			{
				Name:    "kubernetes",
				Version: "1.14.5",
			},
		},
		Name:    "azure-operator",
		Version: "2.4.1",
	}
}
