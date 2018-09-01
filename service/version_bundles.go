package service

import (
	"github.com/giantswarm/versionbundle"

	"github.com/giantswarm/azure-operator/service/controller/v2"
	"github.com/giantswarm/azure-operator/service/controller/v2patch1"
	"github.com/giantswarm/azure-operator/service/controller/v3"
	"github.com/giantswarm/azure-operator/service/controller/v4"
)

func NewVersionBundles() []versionbundle.Bundle {
	var versionBundles []versionbundle.Bundle

	versionBundles = append(versionBundles, v2.VersionBundle())
	versionBundles = append(versionBundles, v2patch1.VersionBundle())
	versionBundles = append(versionBundles, v3.VersionBundle())
	versionBundles = append(versionBundles, v4.VersionBundle())

	return versionBundles
}
