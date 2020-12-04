package setup

import (
	"context"
	"time"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	corev1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/core/v1alpha1"
	"github.com/giantswarm/apiextensions/v3/pkg/crd"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/microerror"
)

func ensureCRDs(ctx context.Context, config Config) error {
	{
		config.Logger.Debugf(ctx, "ensuring appcatalog CRD exists")

		err := config.K8sClients.CRDClient().EnsureCreated(ctx, crd.LoadV1("application.giantswarm.io", "AppCatalog"), backoff.NewMaxRetries(7, 1*time.Second))
		if err != nil {
			return microerror.Mask(err)
		}

		config.Logger.Debugf(ctx, "ensured appcatalog CRD exists")
	}

	{
		config.Logger.Debugf(ctx, "ensuring App CRD exists")

		err := config.K8sClients.CRDClient().EnsureCreated(ctx, v1alpha1.NewAppCRD(), backoff.NewMaxRetries(7, 1*time.Second))
		if err != nil {
			return microerror.Mask(err)
		}

		config.Logger.Debugf(ctx, "ensured App CRD exists")
	}

	{
		config.Logger.Debugf(ctx, "ensuring Chart CRD exists")

		err := config.K8sClients.CRDClient().EnsureCreated(ctx, v1alpha1.NewChartCRD(), backoff.NewMaxRetries(7, 1*time.Second))
		if err != nil {
			return microerror.Mask(err)
		}

		config.Logger.Debugf(ctx, "ensured Chart CRD exists")
	}

	{
		config.Logger.Debugf(ctx, "ensuring Spark CRD exists")

		err := config.K8sClients.CRDClient().EnsureCreated(ctx, corev1alpha1.NewSparkCRD(), backoff.NewMaxRetries(7, 1*time.Second))
		if err != nil {
			return microerror.Mask(err)
		}

		config.Logger.Debugf(ctx, "ensured Spark CRD exists")
	}

	{
		config.Logger.Debugf(ctx, "ensuring CertConfig CRD exists")

		err := config.K8sClients.CRDClient().EnsureCreated(ctx, crd.LoadV1("core.giantswarm.io", "CertConfig"), backoff.NewMaxRetries(7, 1*time.Second))
		if err != nil {
			return microerror.Mask(err)
		}

		config.Logger.Debugf(ctx, "ensured CertConfig CRD exists")
	}

	{
		config.Logger.Debugf(ctx, "ensuring drainerconfig CRD exists")

		err := config.K8sClients.CRDClient().EnsureCreated(ctx, crd.LoadV1("core.giantswarm.io", "DrainerConfig"), backoff.NewMaxRetries(7, 1*time.Second))
		if err != nil {
			return microerror.Mask(err)
		}

		config.Logger.Debugf(ctx, "ensured drainerconfig CRD exists")
	}

	{
		config.Logger.Debugf(ctx, "ensuring storageconfig CRD exists")

		err := config.K8sClients.CRDClient().EnsureCreated(ctx, crd.LoadV1("core.giantswarm.io", "StorageConfig"), backoff.NewMaxRetries(7, 1*time.Second))
		if err != nil {
			return microerror.Mask(err)
		}

		config.Logger.Debugf(ctx, "ensured storageconfig CRD exists")
	}

	return nil
}
