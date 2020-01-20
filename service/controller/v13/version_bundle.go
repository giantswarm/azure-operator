package v12

import (
	"github.com/giantswarm/versionbundle"
)

func VersionBundle() versionbundle.Bundle {
	return versionbundle.Bundle{
		Changelogs: []versionbundle.Changelog{
			{
				Component:   "azure-operator",
				Description: "Fixed automatic reconciliation of failed deployments.",
				Kind:        versionbundle.KindFixed,
			},
			{
				Component:   "azure-operator",
				Description: "Using checksum calculation to avoid applying the same VMSS template over and over again.",
				Kind:        versionbundle.KindChanged,
			},
			{
				Component:   "azure-operator",
				Description: "Fixed the systemd unit that mounts the Docker volume on worker nodes.",
				Kind:        versionbundle.KindFixed,
			},
			{
				Component:   "azure-operator",
				Description: "Added a new volume for the kubelet directory on worker nodes.",
				Kind:        versionbundle.KindAdded,
			},
			{
				Component:   "azure-operator",
				Description: "Hardened 'restricted' PodSecurityPolicy (added UID range).",
				Kind:        versionbundle.KindSecurity,
			},
			{
				Component:   "azure-operator",
				Description: "Added support for multi AZ deployments.",
				Kind:        versionbundle.KindAdded,
			},
			{
				Component:   "kubernetes",
				Description: "Add Deny All as default Network Policy in kube-system and giantswarm namespaces.",
				Kind:        versionbundle.KindChanged,
				URLs: []string{
					"https://github.com/giantswarm/k8scloudconfig/pull/609",
				},
			},
			{
				Component:   "calico",
				Description: "Update from v3.9.1 to v3.10.1.",
				Kind:        versionbundle.KindChanged,
				URLs: []string{
					"https://github.com/giantswarm/azure-operator/pull/609",
				},
			},
			{
				Component:   "containerlinux",
				Description: "Update from v2191.5.0 to v2247.6.0.",
				Kind:        versionbundle.KindChanged,
				URLs: []string{
					"https://github.com/giantswarm/azure-operator/pull/609",
				},
			},
			{
				Component:   "etcd",
				Description: "Update from v3.3.15 to v3.3.17.",
				Kind:        versionbundle.KindChanged,
				URLs: []string{
					"https://github.com/giantswarm/azure-operator/pull/609",
				},
			},
			{
				Component:   "kubernetes",
				Description: "Update from v1.15.5 to v1.16.3.",
				Kind:        versionbundle.KindAdded,
				URLs: []string{
					"https://github.com/giantswarm/azure-operator/pull/609",
				},
			},
		},
		Components: []versionbundle.Component{
			{
				Name:    "calico",
				Version: "3.10.1",
			},
			{
				Name:    "containerlinux",
				Version: "2247.6.0",
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
		Version: "2.8.0",
	}
}
