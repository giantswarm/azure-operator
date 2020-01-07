package v10patch2

import (
	"github.com/giantswarm/versionbundle"
)

func VersionBundle() versionbundle.Bundle {
	return versionbundle.Bundle{
		Changelogs: []versionbundle.Changelog{
			{
				Component:   "kubernetes",
				Description: "Update from v1.14.6 to v1.14.10.",
				Kind:        versionbundle.KindChanged,
				URLs: []string{
					"https://github.com/giantswarm/azure-operator/pull/611",
					"https://github.com/kubernetes/kubernetes/blob/master/CHANGELOG-1.14.md#changelog-since-v11410",
				},
			},
			{
				Component:   "azure-operator",
				Description: "Add new rule to the Public Load Balancer to allow outgoing UDP traffic from the master nodes.",
				Kind:        versionbundle.KindAdded,
				URLs: []string{
					"https://github.com/giantswarm/azure-operator/pull/579",
				},
			},
			{
				Component:   "containerlinux",
				Description: "Increase fs.inotify.max_user_instances to 8192.",
				Kind:        versionbundle.KindChanged,
				URLs: []string{
					"https://github.com/giantswarm/k8scloudconfig/pull/617",
				},
			},
			{
				Component:   "containerlinux",
				Description: "Update from 2135.4.0 to 2135.6.0 for improved regional availability.",
				Kind:        versionbundle.KindChanged,
				URLs: []string{
					"https://github.com/giantswarm/azure-operator/pull/613",
				},
			},
		},
		Components: []versionbundle.Component{
			{
				Name:    "calico",
				Version: "3.8.2",
			},
			{
				Name:    "containerlinux",
				Version: "2135.6.0",
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
				Version: "1.14.10",
			},
		},
		Name:    "azure-operator",
		Version: "2.6.2",
	}
}
