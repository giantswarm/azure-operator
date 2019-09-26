package service

import (
	"github.com/giantswarm/versionbundle"

	v10 "github.com/giantswarm/azure-operator/service/controller/v10"
	"github.com/giantswarm/azure-operator/service/controller/v10patch1"
	v11 "github.com/giantswarm/azure-operator/service/controller/v11"
	v6 "github.com/giantswarm/azure-operator/service/controller/v6"
	v7 "github.com/giantswarm/azure-operator/service/controller/v7"
	v8 "github.com/giantswarm/azure-operator/service/controller/v8"
	"github.com/giantswarm/azure-operator/service/controller/v8patch1"
	v9 "github.com/giantswarm/azure-operator/service/controller/v9"
)

func NewVersionBundles() []versionbundle.Bundle {
	var versionBundles []versionbundle.Bundle

	versionBundles = append(versionBundles, v6.VersionBundle())
	versionBundles = append(versionBundles, v7.VersionBundle())
	versionBundles = append(versionBundles, v8.VersionBundle())
	versionBundles = append(versionBundles, v8patch1.VersionBundle())
	versionBundles = append(versionBundles, v9.VersionBundle())
	versionBundles = append(versionBundles, v10.VersionBundle())
	versionBundles = append(versionBundles, v10patch1.VersionBundle())
	versionBundles = append(versionBundles, v11.VersionBundle())

	return versionBundles
}
