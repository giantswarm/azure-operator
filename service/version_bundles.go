package service

import (
	"time"

	"github.com/giantswarm/versionbundle"
)

func newVersionBundles() []versionbundle.Bundle {
	return []versionbundle.Bundle{
		{
			Changelogs: []versionbundle.Changelog{
				{
					Component:   "kubernetes",
					Description: "Enabled encryption at rest.",
					Kind:        versionbundle.KindAdded,
				},
				{
					Component:   "kubernetes",
					Description: "Updated to version 1.9.2.",
					Kind:        versionbundle.KindChanged,
				},
				{
					Component:   "kubernetes",
					Description: "Switched vanilla (previously coreos) hyperkube image.",
					Kind:        versionbundle.KindChanged,
				},
				{
					Component:   "calico",
					Description: "Updated to version 3.0.1.",
					Kind:        versionbundle.KindChanged,
				},
				{
					Component:   "containerlinux",
					Description: "Updated to version 1576.5.0.",
					Kind:        versionbundle.KindChanged,
				},
				{
					Component:   "docker",
					Description: "Updated to version 17.09.0.",
					Kind:        versionbundle.KindChanged,
				},
				{
					Component:   "calico",
					Description: "Removed calico-ipip-pinger.",
					Kind:        versionbundle.KindRemoved,
				},
				{
					Component:   "calico",
					Description: "Removed calico-node-controller.",
					Kind:        versionbundle.KindRemoved,
				},
				{
					Component:   "cloudconfig",
					Description: "Replace systemd units for Kubernetes components with self-hosted pods.",
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
					Version: "1576.5.0",
				},
				{
					Name:    "docker",
					Version: "17.09.0",
				},
				{
					Name:    "etcd",
					Version: "3.2.7",
				},
				{
					Name:    "coredns",
					Version: "1.0.5",
				},
				{
					Name:    "kubernetes",
					Version: "1.9.2",
				},
				{
					Name:    "nginx-ingress-controller",
					Version: "0.10.2",
				},
			},
			Dependencies: []versionbundle.Dependency{},
			Deprecated:   false,
			Name:         "azure-operator",
			Time:         time.Date(2018, time.January, 7, 8, 35, 0, 0, time.UTC),
			Version:      "0.1.0",
			WIP:          true,
		},
	}
}
