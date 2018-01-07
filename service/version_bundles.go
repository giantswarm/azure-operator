package service

import (
	"time"

	"github.com/giantswarm/versionbundle"
)

func newVersionBundles() []versionbundle.Bundle {
	return []versionbundle.Bundle{
		{
			Changelogs: []versionbundle.Changelog{},
			Components: []versionbundle.Component{
				{
					Name:    "calico",
					Version: "3.0.1",
				},
				{
					Name:    "docker",
					Version: "1.12.6",
				},
				{
					Name:    "etcd",
					Version: "3.2.7",
				},
				{
					Name:    "kubedns",
					Version: "1.14.5",
				},
				{
					Name:    "kubernetes",
					Version: "1.9.0",
				},
				{
					Name:    "nginx-ingress-controller",
					Version: "0.9.0",
				},
			},
			Dependencies: []versionbundle.Dependency{},
			Deprecated:   true,
			Name:         "azure-operator",
			Time:         time.Date(2018, time.January, 7, 8, 35, 0, 0, time.UTC),
			Version:      "0.1.0",
			WIP:          true,
		},
	}
}
