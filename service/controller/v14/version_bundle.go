package v14

import (
	"github.com/giantswarm/versionbundle"
)

const CoreosVersion = "2303.4.0"

func VersionBundle() versionbundle.Bundle {
	return versionbundle.Bundle{
		Changelogs: []versionbundle.Changelog{
			{
				Component:   "azure-operator",
				Description: "TODO",
				Kind:        versionbundle.KindChanged,
				URLs:        []string{},
			},
		},
		Components: []versionbundle.Component{
			{
				Name:    "calico",
				Version: "3.10.1",
			},
			{
				Name:    "containerlinux",
				Version: CoreosVersion,
			},
			{
				Name:    "docker",
				Version: "18.06.1",
			},
			{
				Name:    "etcd",
				Version: "3.3.17",
			},
			{
				Name:    "kubernetes",
				Version: "1.16.3",
			},
		},
		Name:    "azure-operator",
		Version: "2.10.0",
	}
}
