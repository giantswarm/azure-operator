package cmd

import (
	"context"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/giantswarm/e2e-harness/cmd/internal"
	"github.com/giantswarm/e2e-harness/pkg/cluster"
	"github.com/giantswarm/e2e-harness/pkg/docker"
	"github.com/giantswarm/e2e-harness/pkg/harness"
	"github.com/giantswarm/e2e-harness/pkg/patterns"
	"github.com/giantswarm/e2e-harness/pkg/project"
	"github.com/giantswarm/e2e-harness/pkg/tasks"
	"github.com/giantswarm/e2e-harness/pkg/wait"
)

var (
	TeardownCmd = &cobra.Command{
		Use:   "teardown",
		Short: "teardown e2e tests",
		Run:   internal.NewRunFunc(runTeardown),
	}
)

var (
	teardownCmdTestDir string
)

func init() {
	RootCmd.AddCommand(TeardownCmd)

	TeardownCmd.Flags().StringVar(&teardownCmdTestDir, "test-dir", project.DefaultDirectory, "Name of the directory containing executable tests.")
}

func runTeardown(ctx context.Context, cmd *cobra.Command, args []string) error {
	logger, err := micrologger.New(micrologger.Config{})
	if err != nil {
		return microerror.Mask(err)
	}

	fs := afero.NewOsFs()

	h := harness.New(logger, fs, harness.Config{})
	cfg, err := h.ReadConfig()
	if err != nil {
		return err
	}
	projectTag := harness.GetProjectTag()
	projectName := harness.GetProjectName()

	// use latest tag for consumer projects (not dog-fooding e2e-harness)
	e2eHarnessTag := projectTag
	if projectName != "e2e-harness" {
		e2eHarnessTag = "latest"
	}

	var d *docker.Docker
	{
		c := docker.Config{
			Logger: logger,

			Dir:             teardownCmdTestDir,
			ExistingCluster: cfg.ExistingCluster,
			ImageTag:        e2eHarnessTag,
			RemoteCluster:   cfg.RemoteCluster,
		}

		d = docker.New(c)
	}
	pa := patterns.New(logger)
	w := wait.New(logger, d, pa)
	pCfg := &project.Config{
		Name: projectName,
		Tag:  projectTag,
	}
	pDeps := &project.Dependencies{
		Logger: logger,
		Runner: d,
		Wait:   w,
		Fs:     fs,
	}
	p := project.New(pDeps, pCfg)

	clusterCfg := cluster.Config{
		Logger:          logger,
		Fs:              fs,
		ExistingCluster: existingCluster,
		RemoteCluster:   remoteCluster,
		Runner:          d,
	}
	c := cluster.New(clusterCfg)

	bundle := []tasks.Task{}

	if cfg.RemoteCluster && !cfg.ExistingCluster {
		bundle = append(bundle, c.Delete)
	} else if !cfg.ExistingCluster {
		bundle = append(bundle, p.CommonTearDownSteps)
	}

	return tasks.Run(ctx, bundle)
}
