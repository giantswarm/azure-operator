package release

import (
	"context"
	"fmt"
	"strings"

	"github.com/giantswarm/apiextensions/pkg/apis/release/v1alpha1"
	"github.com/giantswarm/microerror"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/azure-operator/v3/pkg/label"
	"github.com/giantswarm/azure-operator/v3/service/controller/controllercontext"
	"github.com/giantswarm/azure-operator/v3/service/controller/key"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return microerror.Mask(err)
	}
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	var release *v1alpha1.Release
	{
		releaseVersion := cr.Labels[label.ReleaseVersion]
		if !strings.HasPrefix(releaseVersion, "v") {
			releaseVersion = fmt.Sprintf("v%s", releaseVersion)
		}

		release, err = r.g8sClient.ReleaseV1alpha1().Releases().Get(releaseVersion, metav1.GetOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
	}

	cc.Release.Components = release.Spec.Components

	return nil
}
