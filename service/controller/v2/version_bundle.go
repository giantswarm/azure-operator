package v2

import (
	"time"

	"github.com/giantswarm/versionbundle"
)

func VersionBundle() versionbundle.Bundle {
	return versionbundle.Bundle{
		Changelogs: []versionbundle.Changelog{
			{
				Component:   "kubernetes",
				Description: "updated to 1.10.2.",
				Kind:        versionbundle.KindChanged,
			},
			{
				Component:   "cloudconfig",
				Description: "Removed kube-state-metrics related components (will be managed by chart-operator).",
				Kind:        versionbundle.KindRemoved,
			},
			{
				Component:   "azure-operator",
				Description: "Added German cloud support.",
				Kind:        versionbundle.KindAdded,
			},
			{
				Component:   "cloudconfig",
				Description: "Updated Calico to 3.0.5",
				Kind:        versionbundle.KindChanged,
			},
			{
				Component:   "azure-operator",
				Description: "Add etcd private loadbalancer.",
				Kind:        versionbundle.KindChanged,
			},
		},
		Components: []versionbundle.Component{
			{
				Name:    "calico",
				Version: "3.0.5",
			},
			{
				Name:    "containerlinux",
				Version: "1688.5.3",
			},
			{
				Name:    "docker",
				Version: "17.12.1",
			},
			{
				Name:    "etcd",
				Version: "3.3.3",
			},
			{
				Name:    "coredns",
				Version: "1.1.1",
			},
			{
				Name:    "kubernetes",
				Version: "1.10.2",
			},
			{
				Name:    "nginx-ingress-controller",
				Version: "0.12.0",
			},
		},
		Dependencies: []versionbundle.Dependency{},
		Deprecated:   false,
		Name:         "azure-operator",
		Time:         time.Date(2018, time.May, 14, 12, 00, 0, 0, time.UTC),
		Version:      "0.2.0",
		WIP:          true,
	}
}
