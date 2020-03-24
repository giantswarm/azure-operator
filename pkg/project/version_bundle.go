package project

import (
	"github.com/giantswarm/versionbundle"

	"github.com/giantswarm/azure-operator/service/controller/key"
)

func NewVersionBundle() versionbundle.Bundle {
	return versionbundle.Bundle{
		Changelogs: []versionbundle.Changelog{
			{
				Component:   Name(),
				Description: "Fix workers' overprovisioning during cluster creation.",
				Kind:        versionbundle.KindFixed,
				URLs:        []string{"https://github.com/giantswarm/azure-operator/pull/679"},
			},
		},
		Components: []versionbundle.Component{
			{
				Name:    "calico",
				Version: "3.10.1",
			},
			{
				Name:    "containerlinux",
				Version: key.CoreosVersion,
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
		Name:    Name(),
		Version: Version(),
	}
}
