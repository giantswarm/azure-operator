package service

import (
	"github.com/giantswarm/versionbundle"

	"github.com/giantswarm/azure-operator/service/controller/v4patch1"
	"github.com/giantswarm/azure-operator/service/controller/v4patch2"
	"github.com/giantswarm/azure-operator/service/controller/v5"
	"github.com/giantswarm/azure-operator/service/controller/v6"
	"github.com/giantswarm/azure-operator/service/controller/v7"
)

func NewVersionBundles() []versionbundle.Bundle {
	var versionBundles []versionbundle.Bundle

	versionBundles = append(versionBundles, v4patch1.VersionBundle())
	versionBundles = append(versionBundles, v4patch2.VersionBundle())
	versionBundles = append(versionBundles, v5.VersionBundle())
	versionBundles = append(versionBundles, v6.VersionBundle())
	versionBundles = append(versionBundles, v7.VersionBundle())

	return versionBundles
}
