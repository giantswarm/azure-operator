package legacyresource

import (
	"context"
	"fmt"
	"time"

	"github.com/giantswarm/backoff"
	"github.com/giantswarm/helmclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"k8s.io/helm/pkg/helm"
)

const (
	defaultNamespace = "default"
)

type Config struct {
	HelmClient *helmclient.Client
	Logger     micrologger.Logger

	Namespace string
}

type Resource struct {
	helmClient *helmclient.Client
	logger     micrologger.Logger

	namespace string
}

func New(config Config) (*Resource, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.HelmClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.HelmClient must not be empty", config)
	}
	if config.Namespace == "" {
		config.Namespace = defaultNamespace
	}
	c := &Resource{
		helmClient: config.HelmClient,
		logger:     config.Logger,

		namespace: config.Namespace,
	}

	return c, nil
}

func (r *Resource) Delete(name string) error {
	ctx := context.TODO()

	err := r.helmClient.DeleteRelease(ctx, name, helm.DeletePurge(true))
	if helmclient.IsReleaseNotFound(err) {
		return microerror.Maskf(releaseNotFoundError, name)
	} else if helmclient.IsTillerNotFound(err) {
		return microerror.Mask(tillerNotFoundError)
	} else if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *Resource) EnsureDeleted(ctx context.Context, name string) error {
	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensuring deletion of release %#q", name))

	err := r.helmClient.DeleteRelease(ctx, name, helm.DeletePurge(true))
	if helmclient.IsReleaseNotFound(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("release %#q does not exist", name))
	} else if helmclient.IsTillerNotFound(err) {
		r.logger.LogCtx(ctx, "level", "warning", "message", "tiller is not found/installed")
	} else if err != nil {
		return microerror.Mask(err)
	} else {
		r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("deleted release %#q", name))
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensured deletion of release %#q", name))

	return nil
}

func (r *Resource) Install(name, url, values string, conditions ...func() error) error {
	ctx := context.TODO()

	tarballPath, err := r.helmClient.PullChartTarball(ctx, url)
	defer func() {
		fs := afero.NewOsFs()
		err := fs.Remove(tarballPath)
		if err != nil {
			r.logger.LogCtx(ctx, "level", "error", "message", "failed to delete tarball", "stack", fmt.Sprintf("%#v", err))
		}
	}()
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.helmClient.InstallReleaseFromTarball(ctx, tarballPath, r.namespace, helm.ReleaseName(name), helm.ValueOverrides([]byte(values)), helm.InstallWait(true))
	if err != nil {
		return microerror.Mask(err)
	}

	for _, c := range conditions {
		err = backoff.Retry(c, backoff.NewExponential(backoff.ShortMaxWait, backoff.ShortMaxInterval))
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

func (r *Resource) Update(name, url, values string, conditions ...func() error) error {
	ctx := context.TODO()

	tarballPath, err := r.helmClient.PullChartTarball(ctx, url)
	defer func() {
		fs := afero.NewOsFs()
		err := fs.Remove(tarballPath)
		if err != nil {
			r.logger.LogCtx(ctx, "level", "error", "message", "failed to delete tarball", "stack", fmt.Sprintf("%#v", err))
		}
	}()
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.helmClient.UpdateReleaseFromTarball(ctx, name, tarballPath, helm.UpdateValueOverrides([]byte(values)))
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *Resource) WaitForStatus(release string, status string) error {
	ctx := context.TODO()

	operation := func() error {
		rc, err := r.helmClient.GetReleaseContent(ctx, release)
		if helmclient.IsReleaseNotFound(err) && status == "DELETED" {
			// Error is expected because we purge releases when deleting.
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}
		if rc.Status != status {
			return microerror.Maskf(releaseStatusNotMatchingError, "waiting for '%s', current '%s'", status, rc.Status)
		}
		return nil
	}

	notify := func(err error, t time.Duration) {
		r.logger.Log("level", "debug", "message", fmt.Sprintf("failed to get release status '%s': retrying in %s", status, t), "stack", fmt.Sprintf("%v", err))
	}

	b := backoff.NewExponential(backoff.MediumMaxWait, backoff.LongMaxInterval)
	err := backoff.RetryNotify(operation, b, notify)
	if err != nil {
		return microerror.Mask(err)
	}
	return nil
}

func (r *Resource) WaitForVersion(release string, version string) error {
	ctx := context.TODO()

	operation := func() error {
		rh, err := r.helmClient.GetReleaseHistory(ctx, release)
		if err != nil {
			return microerror.Mask(err)
		}
		if rh.Version != version {
			return microerror.Maskf(releaseVersionNotMatchingError, "waiting for '%s', current '%s'", version, rh.Version)
		}
		return nil
	}

	notify := func(err error, t time.Duration) {
		r.logger.Log("level", "debug", "message", fmt.Sprintf("failed to get release version '%s': retrying in %s", version, t), "stack", fmt.Sprintf("%v", err))
	}

	b := backoff.NewExponential(backoff.ShortMaxWait, backoff.LongMaxInterval)
	err := backoff.RetryNotify(operation, b, notify)
	if err != nil {
		return microerror.Mask(err)
	}
	return nil
}
