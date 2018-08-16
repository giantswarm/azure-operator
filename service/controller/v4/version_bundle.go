package v4

import (
	"github.com/giantswarm/versionbundle"
)

func VersionBundle() versionbundle.Bundle {
	return versionbundle.Bundle{
		Changelogs: []versionbundle.Changelog{
			{
				Component:   "azure-operator",
				Description: "Added CA public key into trusted user keys for SSO ssh.",
				Kind:        versionbundle.KindAdded,
			},
			{
				Component:   "cloudconfig",
				Description: "Added support for status subresources for CRDs.",
				Kind:        versionbundle.KindAdded,
			},
			{
				Component:   "azure-operator",
				Description: "Added support for etcd monitoring.",
				Kind:        versionbundle.KindAdded,
			},
			{
				Component:   "azure-operator",
				Description: "Fixed Azure disk mounting.",
				Kind:        versionbundle.KindChanged,
			},
		},
		Components: []versionbundle.Component{
			{
				Name:    "calico",
				Version: "3.0.8",
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
				Name:    "coredns",
				Version: "1.1.1",
			},
			{
				Name:    "kubernetes",
				Version: "1.10.4",
			},
		},
		Name:    "azure-operator",
		Version: "1.2.0",
	}
}
