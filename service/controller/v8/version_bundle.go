package v8

import (
	"github.com/giantswarm/versionbundle"
)

func VersionBundle() versionbundle.Bundle {
	return versionbundle.Bundle{
		Changelogs: []versionbundle.Changelog{
			{
				Component:   "kubernetes",
				Description: "Update kubernetes to 1.14.3. More info here: https://github.com/kubernetes/kubernetes/blob/master/CHANGELOG-1.14.md",
				Kind:        versionbundle.KindChanged,
			},
			{
				Component:   "calico",
				Description: "Update calico to 3.7.2. More info here: https://docs.projectcalico.org/v3.7/release-notes/",
				Kind:        versionbundle.KindChanged,
			},
			{
				Component:   "containerlinux",
				Description: "Update to 2135.4.0. More info here: https://github.com/coreos/manifest/releases/tag/v2135.4.0",
				Kind:        versionbundle.KindChanged,
			},
			{
				Component:   "etcd",
				Description: "Update to 3.3.13. More info here: https://github.com/etcd-io/etcd/blob/master/CHANGELOG-3.3.md#v3313-2019-05-02",
				Kind:        versionbundle.KindChanged,
			},
		},
		Components: []versionbundle.Component{
			{
				Name:    "calico",
				Version: "3.7.2",
			},
			{
				Name:    "containerlinux",
				Version: "2135.4.0",
			},
			{
				Name:    "docker",
				Version: "18.06.1",
			},
			{
				Name:    "etcd",
				Version: "3.3.13",
			},
			{
				Name:    "kubernetes",
				Version: "1.14.3",
			},
		},
		Name:    "azure-operator",
		Version: "2.4.0",
	}
}
