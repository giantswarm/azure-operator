package v4patch2

import (
	"github.com/giantswarm/versionbundle"
)

func VersionBundle() versionbundle.Bundle {
	return versionbundle.Bundle{
		Changelogs: []versionbundle.Changelog{
			{
				Component:   "containerlinux",
				Description: "Fix for CVE-2019-5736.",
				Kind:        versionbundle.KindSecurity,
			},
		},
		Components: []versionbundle.Component{
			{
				Name:    "calico",
				Version: "3.2.0",
			},
			{
				Name:    "containerlinux",
				Version: "1967.5.0",
			},
			{
				Name:    "docker",
				Version: "18.06.1",
			},
			{
				Name:    "etcd",
				Version: "3.3.3",
			},
			{
				Name:    "kubernetes",
				Version: "1.11.5",
			},
		},
		Name:    "azure-operator",
		Version: "2.0.2",
	}
}
