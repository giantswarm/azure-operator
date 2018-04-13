package v1

import (
	"time"

	"github.com/giantswarm/versionbundle"
)

func VersionBundle() versionbundle.Bundle {
	return versionbundle.Bundle{
		Changelogs: []versionbundle.Changelog{
			{
				Component:   "kubernetes",
				Description: "Updated to version 1.10.1.",
				Kind:        versionbundle.KindChanged,
			},
			{
				Component:   "containerlinux",
				Description: "Updated to version 1632.3.0.",
				Kind:        versionbundle.KindChanged,
			},
			{
				Component:   "docker",
				Description: "Updated to version 17.09.0.",
				Kind:        versionbundle.KindChanged,
			},
			{
				Component:   "azure-operator",
				Description: "Change default etcd data dir to /var/lib/etcd.",
				Kind:        versionbundle.KindChanged,
			},
		},
		Components: []versionbundle.Component{
			{
				Name:    "calico",
				Version: "3.0.1",
			},
			{
				Name:    "containerlinux",
				Version: "1632.3.0",
			},
			{
				Name:    "docker",
				Version: "17.09.0",
			},
			{
				Name:    "etcd",
				Version: "3.3.1",
			},
			{
				Name:    "coredns",
				Version: "1.0.6",
			},
			{
				Name:    "kubernetes",
				Version: "1.10.1",
			},
			{
				Name:    "nginx-ingress-controller",
				Version: "0.12.0",
			},
		},
		Dependencies: []versionbundle.Dependency{},
		Deprecated:   false,
		Name:         "azure-operator",
		Time:         time.Date(2018, time.January, 7, 8, 35, 0, 0, time.UTC),
		Version:      "0.1.0",
		WIP:          true,
	}
}
