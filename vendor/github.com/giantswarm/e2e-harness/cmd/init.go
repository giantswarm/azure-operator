package cmd

import (
	"context"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/giantswarm/e2e-harness/cmd/internal"
	"github.com/giantswarm/e2e-harness/pkg/harness"
	"github.com/giantswarm/e2e-harness/pkg/initializer"
	"github.com/giantswarm/e2e-harness/pkg/tasks"
)

var (
	InitCmd = &cobra.Command{
		Use:   "init",
		Short: "initialize project to develop and run k8s e2e tests",
		Run:   internal.NewRunFunc(runInit),
	}
)

func init() {
	RootCmd.AddCommand(InitCmd)
}

func runInit(ctx context.Context, cmd *cobra.Command, args []string) error {
	logger, err := micrologger.New(micrologger.Config{})
	if err != nil {
		return microerror.Mask(err)
	}

	projectName := harness.GetProjectName()
	fs := afero.NewOsFs()
	i := initializer.New(logger, fs, projectName)

	// tasks to run.
	bundle := []tasks.Task{
		i.CreateLayout,
	}

	return tasks.Run(ctx, bundle)
}
