package release

import (
	"context"
	"fmt"
	"strings"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/release/v1alpha1"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v5/pkg/label"
	"github.com/giantswarm/azure-operator/v5/service/controller/controllercontext"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	m, err := meta.Accessor(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "finding release from cr")

	var release v1alpha1.Release
	{
		releaseVersion := m.GetLabels()[label.ReleaseVersion]
		if !strings.HasPrefix(releaseVersion, "v") {
			releaseVersion = fmt.Sprintf("v%s", releaseVersion)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found release version %q from cr", releaseVersion))

		r.logger.LogCtx(ctx, "level", "debug", "message", "reading release object")
		err = r.k8sClient.CtrlClient().Get(ctx, client.ObjectKey{Namespace: corev1.NamespaceAll, Name: releaseVersion}, &release)
		if err != nil {
			return microerror.Mask(err)
		}
		r.logger.LogCtx(ctx, "level", "debug", "message", "read release object")
	}

	cc.Release.Release = release

	r.logger.LogCtx(ctx, "level", "debug", "message", "saved release object in controllercontext")

	return nil
}
