package service

import (
	"github.com/giantswarm/versionbundle"

	"github.com/giantswarm/azure-operator/service/controller/v6"
	"github.com/giantswarm/azure-operator/service/controller/v7"
	"github.com/giantswarm/azure-operator/service/controller/v8"
	"github.com/giantswarm/azure-operator/service/controller/v8patch1"
	"github.com/giantswarm/azure-operator/service/controller/v10"
)

func NewVersionBundles() []versionbundle.Bundle {
	var versionBundles []versionbundle.Bundle

	versionBundles = append(versionBundles, v6.VersionBundle())
	versionBundles = append(versionBundles, v7.VersionBundle())
	versionBundles = append(versionBundles, v8.VersionBundle())
	versionBundles = append(versionBundles, v8patch1.VersionBundle())
	versionBundles = append(versionBundles, v10.VersionBundle())

	return versionBundles
}
