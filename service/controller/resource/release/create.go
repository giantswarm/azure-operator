package release

import (
	"context"
	"fmt"
	"strings"

	"github.com/giantswarm/apiextensions/pkg/apis/release/v1alpha1"
	"github.com/giantswarm/microerror"
	"sigs.k8s.io/controller-runtime/pkg/client"

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

	var release v1alpha1.Release
	{
		releaseVersion := cr.Labels[label.ReleaseVersion]
		if !strings.HasPrefix(releaseVersion, "v") {
			releaseVersion = fmt.Sprintf("v%s", releaseVersion)
		}

		err = r.k8sClient.CtrlClient().Get(ctx, client.ObjectKey{Namespace: "", Name: releaseVersion}, &release)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	cc.Release.Release = release

	return nil
}
