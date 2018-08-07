package update

import (
	"context"
	"fmt"
	"time"

	"github.com/giantswarm/backoff"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/e2etests/update/provider"
)

type Config struct {
	Logger   micrologger.Logger
	Provider provider.Interface
}

type Update struct {
	logger   micrologger.Logger
	provider provider.Interface
}

func New(config Config) (*Update, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.Provider == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Provider must not be empty", config)
	}

	u := &Update{
		logger:   config.Logger,
		provider: config.Provider,
	}

	return u, nil
}

func (u *Update) Test(ctx context.Context) error {
	var err error

	var currentVersion string
	{
		u.logger.LogCtx(ctx, "level", "debug", "message", "looking for the current version bundle version")

		currentVersion, err = u.provider.CurrentVersion()
		if provider.IsVersionNotFound(err) {
			u.logger.LogCtx(ctx, "level", "debug", "message", "did not find the current version bundle version")
			u.logger.LogCtx(ctx, "level", "debug", "message", "canceling e2e test for current version")
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}

		u.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found the current version bundle version '%s'", currentVersion))
	}

	var nextVersion string
	{
		u.logger.LogCtx(ctx, "level", "debug", "message", "looking for the next version bundle version")

		nextVersion, err = u.provider.NextVersion()
		if provider.IsVersionNotFound(err) {
			u.logger.LogCtx(ctx, "level", "debug", "message", "did not find the next version bundle version")
			u.logger.LogCtx(ctx, "level", "debug", "message", "canceling e2e test for current version")
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}

		u.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found the next version bundle version '%s'", nextVersion))
	}

	{
		u.logger.LogCtx(ctx, "level", "debug", "message", "updating the guest cluster with the new version bundle version")

		err := u.provider.UpdateVersion(nextVersion)
		if err != nil {
			return microerror.Mask(err)
		}

		u.logger.LogCtx(ctx, "level", "debug", "message", "updated the guest cluster with the new version bundle version")
	}

	{
		u.logger.LogCtx(ctx, "level", "debug", "message", "updating the guest cluster with the new version bundle version")

		o := func() error {
			isUpdated, err := u.provider.IsUpdated()
			if err != nil {
				return microerror.Mask(err)
			}
			if isUpdated {
				return backoff.Permanent(alreadyUpdatedError)
			}

			return nil
		}
		b := backoff.NewConstant(60*time.Minute, 5*time.Minute)
		n := backoff.NewNotifier(u.logger, ctx)

		err := backoff.RetryNotify(o, b, n)
		if IsAlreadyUpdated(err) {
			// fall through
		} else if err != nil {
			return microerror.Mask(err)
		}

		u.logger.LogCtx(ctx, "level", "debug", "message", "updated the guest cluster with the new version bundle version")
	}

	return nil
}
