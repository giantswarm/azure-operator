package release

import (
	"context"
	"fmt"
	"strings"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/release-operator/v3/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v6/pkg/label"
	"github.com/giantswarm/azure-operator/v6/service/controller/controllercontext"
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

	r.logger.Debugf(ctx, "finding release from cr")

	var release v1alpha1.Release
	{
		releaseVersion := m.GetLabels()[label.ReleaseVersion]
		if !strings.HasPrefix(releaseVersion, "v") {
			releaseVersion = fmt.Sprintf("v%s", releaseVersion)
		}

		r.logger.Debugf(ctx, "found release version %q from cr", releaseVersion)

		r.logger.Debugf(ctx, "reading release object")
		err = r.k8sClient.CtrlClient().Get(ctx, client.ObjectKey{Namespace: corev1.NamespaceAll, Name: releaseVersion}, &release)
		if err != nil {
			return microerror.Mask(err)
		}
		r.logger.Debugf(ctx, "read release object")
	}

	cc.Release.Release = release

	r.logger.Debugf(ctx, "saved release object in controllercontext")

	return nil
}
