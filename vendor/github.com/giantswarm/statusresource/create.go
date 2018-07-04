package statusresource

import (
	"context"
	"encoding/json"
	"time"

	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return microerror.Mask(err)
	}
	clusterStatus, err := r.clusterStatusFunc(obj)
	if err != nil {
		return microerror.Mask(err)
	}
	currentVersion := clusterStatus.LatestVersion()
	desiredVersion, err := r.versionBundleVersionFunc(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	{
		var patches []Patch

		if clusterStatus.Conditions == nil {
			patches = append(patches, Patch{
				Op:    "add",
				Path:  "/status/cluster/conditions",
				Value: []providerv1alpha1.StatusClusterCondition{},
			})
		}

		if clusterStatus.Versions == nil {
			patches = append(patches, Patch{
				Op:    "add",
				Path:  "/status/cluster/versions",
				Value: []providerv1alpha1.StatusClusterVersion{},
			})
		}

		err := r.patchObject(ctx, accessor, patches)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	// We add the desired guest cluster version to the status history if it is not
	// tracked already. This indicates an update is about to be processed. So we
	// also set the status condition indicating the guest cluster is updating now.
	{
		var patches []Patch

		if currentVersion != "" && currentVersion != desiredVersion {
			patches = append(patches, Patch{
				Op:   "add",
				Path: "/status/cluster/conditions/-",
				Value: providerv1alpha1.StatusClusterCondition{
					Status: providerv1alpha1.StatusClusterStatusTrue,
					Type:   providerv1alpha1.StatusClusterTypeUpdating,
				},
			})

			patches = append(patches, Patch{
				Op:   "add",
				Path: "/status/cluster/versions/-",
				Value: providerv1alpha1.StatusClusterVersion{
					Date:   time.Now(),
					Semver: desiredVersion,
				},
			})
		}

		err = r.patchObject(ctx, accessor, patches)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	// TODO remove this once the transition period is completed and all stati
	// contain the latest version.
	{
		var patches []Patch

		if currentVersion == "" {
			patches = append(patches, Patch{
				Op:   "add",
				Path: "/status/cluster/versions/-",
				Value: providerv1alpha1.StatusClusterVersion{
					Date:   time.Now(),
					Semver: desiredVersion,
				},
			})
		}

		err := r.patchObject(ctx, accessor, patches)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	// TODO when updating state is set and guest cluster is updated set updated status
	// TODO emit metrics when update did not complete within a certain timeframe

	return nil
}

func (r *Resource) patchObject(ctx context.Context, accessor metav1.Object, patches []Patch) error {
	patches = append(patches, Patch{
		Op:    "test",
		Value: accessor.GetResourceVersion(),
		Path:  "/metadata/resourceVersion",
	})

	b, err := json.Marshal(patches)
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.restClient.Patch(types.JSONPatchType).AbsPath(accessor.GetSelfLink()).Body(b).Do().Error()
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
