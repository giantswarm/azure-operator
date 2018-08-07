package v2patch1

import (
	"github.com/giantswarm/versionbundle"
)

func VersionBundle() versionbundle.Bundle {
	return versionbundle.Bundle{
		Changelogs: []versionbundle.Changelog{
			{
				Component:   "kubernetes",
				Description: "Updated to 1.10.4.",
				Kind:        versionbundle.KindChanged,
			},
			{
				Component:   "azure-operator",
				Description: "Added German cloud support.",
				Kind:        versionbundle.KindAdded,
			},
			{
				Component:   "cloudconfig",
				Description: "Updated Calico to 3.0.8",
				Kind:        versionbundle.KindChanged,
			},
			{
				Component:   "azure-operator",
				Description: "Added etcd private loadbalancer.",
				Kind:        versionbundle.KindAdded,
			},
			{
				Component:   "azure-operator",
				Description: "Updated resource implementations to be non-blocking.",
				Kind:        versionbundle.KindChanged,
			},
			{
				Component:   "azure-operator",
				Description: "Added basic support for updates.",
				Kind:        versionbundle.KindChanged,
			},
			{
				Component:   "azure-operator",
				Description: "Ingress load balancer managed by Kubernetes.",
				Kind:        versionbundle.KindChanged,
			},
			{
				Component:   "cloudconfig",
				Description: "Removed Ingress Controller, kube-state-metrics and node-exporter related components (will be managed by chart-operator).",
				Kind:        versionbundle.KindRemoved,
			},
			{
				Component:   "azure-operator",
				Description: "Updated Container Linux to 1745.7.0",
				Kind:        versionbundle.KindChanged,
			},
			{
				Component:   "azure-operator",
				Description: "Add second disk for master vmss.",
				Kind:        versionbundle.KindChanged,
			},
			{
				Component:   "azure-operator",
				Description: "Disable overprovisioning of master vmss.",
				Kind:        versionbundle.KindChanged,
			},
			{
				Component:   "azure-operator",
				Description: "Multiple fixes related to resource reconciliation.",
				Kind:        versionbundle.KindChanged,
			},
			{
				Component:   "azure-operator",
				Description: "Add OIDC support.",
				Kind:        versionbundle.KindAdded,
			},
			{
				Component:   "azure-operator",
				Description: "Add non-MSI support.",
				Kind:        versionbundle.KindAdded,
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
		Version: "1.0.0",
	}
}
