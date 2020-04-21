package project

import (
	"github.com/giantswarm/versionbundle"
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
			{
				Component:   Name(),
				Description: "Modified to retrieve component versions from releases.",
				Kind:        versionbundle.KindChanged,
				URLs:        []string{"https://github.com/giantswarm/azure-operator/pull/727"},
			},
			{
				Component:   Name(),
				Description: "Use Release.Revision in annotation for Helm 3 compatability.",
				Kind:        versionbundle.KindChanged,
				URLs:        []string{"https://github.com/giantswarm/azure-operator/pull/776"},
			},
		},
		Name:    Name(),
		Version: Version(),
	}
}
