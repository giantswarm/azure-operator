package service

import (
	"github.com/giantswarm/versionbundle"

	"github.com/giantswarm/azure-operator/service/controller/v10patch1"
	v11 "github.com/giantswarm/azure-operator/service/controller/v11"
	v12 "github.com/giantswarm/azure-operator/service/controller/v12"
	v13 "github.com/giantswarm/azure-operator/service/controller/v13"
	v7 "github.com/giantswarm/azure-operator/service/controller/v7"
)

func NewVersionBundles() []versionbundle.Bundle {
	var versionBundles []versionbundle.Bundle

	versionBundles = append(versionBundles, v7.VersionBundle())
	versionBundles = append(versionBundles, v10patch1.VersionBundle())
	versionBundles = append(versionBundles, v11.VersionBundle())
	versionBundles = append(versionBundles, v12.VersionBundle())
	versionBundles = append(versionBundles, v13.VersionBundle())

	return versionBundles
}
