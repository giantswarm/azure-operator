package v6

import (
	"github.com/giantswarm/versionbundle"
)

func VersionBundle() versionbundle.Bundle {
	return versionbundle.Bundle{
		Changelogs: []versionbundle.Changelog{
			{
				Component:   "kubernetes",
				Description: "Update Kubernetes to 1.12.3. More info here: https://github.com/kubernetes/kubernetes/blob/master/CHANGELOG-1.12.md",
				Kind:        versionbundle.KindChanged,
			},
			{
				Component:   "calico",
				Description: "Updated to 3.2.3. Also the manifest has proper resource limits and priority class to get QoS policy guaranteed.",
				Kind:        versionbundle.KindChanged,
			},
			{
				Component:   "kubernetes",
				Description: "Enabled admission plugins: DefaultTolerationSeconds, MutatingAdmissionWebhook, ValidatingAdmissionWebhook.",
				Kind:        versionbundle.KindChanged,
			},
			{
				Component:   "container-linux",
				Description: "Updated to latest stable 1855.5.0",
				Kind:        versionbundle.KindChanged,
			},
			{
				Component:   "etcd",
				Description: "Updated to 3.3.9",
				Kind:        versionbundle.KindChanged,
			},
			{
				Component:   "docker",
				Description: "Updated to 18.06.1",
				Kind:        versionbundle.KindChanged,
			},
			{
				Component:   "kube-proxy",
				Description: "Several configuration fixes and it now gets installed and upgraded before Calico.",
				Kind:        versionbundle.KindChanged,
			},
			{
				Component:   "ignition",
				Description: "Updated k8scloudconfig to 4.1.0",
				Kind:        versionbundle.KindChanged,
			},
			{
				Component:   "node-operator",
				Description: "Improved node draining during updates and scaling.",
				Kind:        versionbundle.KindChanged,
			},
			{
				Component:   "kubernetes",
				Description: "Improved Audit policy to reduce the amount of Audit logs (high-volume and low-risk).",
				Kind:        versionbundle.KindChanged,
			},
		},
		Components: []versionbundle.Component{
			{
				Name:    "calico",
				Version: "3.5.1",
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
				Version: "3.3.12",
			},
			{
				Name:    "kubernetes",
				Version: "1.13.3",
			},
		},
		Name:    "azure-operator",
		Version: "2.1.0",
	}
}
