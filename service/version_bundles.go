package service

import (
	"github.com/giantswarm/versionbundle"

	"github.com/giantswarm/azure-operator/service/controller/v4patch1"
	"github.com/giantswarm/azure-operator/service/controller/v4patch2"
	v5 "github.com/giantswarm/azure-operator/service/controller/v5"
	v6 "github.com/giantswarm/azure-operator/service/controller/v6"
	v7 "github.com/giantswarm/azure-operator/service/controller/v7"
	v8 "github.com/giantswarm/azure-operator/service/controller/v8"
)

func NewVersionBundles() []versionbundle.Bundle {
	var versionBundles []versionbundle.Bundle

	versionBundles = append(versionBundles, v4patch1.VersionBundle())
	versionBundles = append(versionBundles, v4patch2.VersionBundle())
	versionBundles = append(versionBundles, v5.VersionBundle())
	versionBundles = append(versionBundles, v6.VersionBundle())
	versionBundles = append(versionBundles, v7.VersionBundle())
	versionBundles = append(versionBundles, v8.VersionBundle())

	return versionBundles
}
