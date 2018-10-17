package v4

import (
	"github.com/giantswarm/versionbundle"
)

func VersionBundle() versionbundle.Bundle {
	return versionbundle.Bundle{
		Changelogs: []versionbundle.Changelog{
			{
				Component:   "kubernetes",
				Description: "Updated to 1.11.1.",
				Kind:        versionbundle.KindChanged,
			},
			{
				Component:   "cloudconfig",
				Description: "Calico updated to version 3.2.0",
				Kind:        versionbundle.KindChanged,
			},
			{
				Component:   "cloudconfig",
				Description: "Removed CoreDNS related components (now managed by chart-operator).",
				Kind:        versionbundle.KindRemoved,
			},
			{
				Component:   "kubernetes",
				Description: "Patched Kubernetes v1.11.1 with apimachinery fix to plug apiserver memory leak.",
				Kind:        versionbundle.KindChanged,
			},
		},
		Components: []versionbundle.Component{
			{
				Name:    "calico",
				Version: "3.2.0",
			},
			{
				Name:    "containerlinux",
				Version: "1745.7.0",
			},
			{
				Name:    "docker",
				Version: "18.03.1",
			},
			{
				Name:    "etcd",
				Version: "3.3.3",
			},
			{
				Name:    "kubernetes",
				Version: "1.11.1",
			},
		},
		Name:    "azure-operator",
		Version: "2.0.0",
	}
}
