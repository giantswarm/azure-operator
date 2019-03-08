package service

import (
	"github.com/giantswarm/versionbundle"

	"github.com/giantswarm/azure-operator/service/controller/v4patch1"
	"github.com/giantswarm/azure-operator/service/controller/v4patch2"
	v5 "github.com/giantswarm/azure-operator/service/controller/v5"
	v5patch1 "github.com/giantswarm/azure-operator/service/controller/v5patch1"
	v6 "github.com/giantswarm/azure-operator/service/controller/v6"
)

func NewVersionBundles() []versionbundle.Bundle {
	var versionBundles []versionbundle.Bundle

	versionBundles = append(versionBundles, v4patch1.VersionBundle())
	versionBundles = append(versionBundles, v4patch2.VersionBundle())
	versionBundles = append(versionBundles, v5.VersionBundle())
	versionBundles = append(versionBundles, v5patch1.VersionBundle())
	versionBundles = append(versionBundles, v6.VersionBundle())

	return versionBundles
}
