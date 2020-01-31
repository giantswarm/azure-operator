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
	SetupCmd = &cobra.Command{
		Use:   "setup",
		Short: "setup e2e tests",
		Run:   internal.NewRunFunc(runSetup),
	}
)

var (
	setupCmdTestDir string
	name            string
	existingCluster bool
	k8sApiUrl       string
	k8sCert         string
	k8sCertCA       string
	k8sCertKey      string
	k8sContext      string
	remoteCluster   bool
)

func init() {
	RootCmd.AddCommand(SetupCmd)

	SetupCmd.Flags().StringVar(&setupCmdTestDir, "test-dir", project.DefaultDirectory, "Name of the directory containing executable tests.")
	SetupCmd.Flags().StringVar(&name, "name", "e2e-harness", "CI execution identifier")
	SetupCmd.Flags().BoolVar(&existingCluster, "existing", false, "can be used with --remote=true to use already existing cluster")
	SetupCmd.Flags().StringVar(&k8sApiUrl, "k8s-api-url", "", "k8s api url for existing cluster")
	SetupCmd.Flags().StringVar(&k8sCert, "k8s-cert", "", "k8s cert for auth for existing cluster")
	SetupCmd.Flags().StringVar(&k8sCertCA, "k8s-cert-ca", "", "k8s cert ca for auth for existing cluster")
	SetupCmd.Flags().StringVar(&k8sContext, "k8s-context", "minikube", "k8s context to use")
	SetupCmd.Flags().StringVar(&k8sCertKey, "k8s-cert-key", "", "k8s cert key for auth for existing cluster")
	SetupCmd.Flags().BoolVar(&remoteCluster, "remote", true, "use remote cluster")
}

func runSetup(ctx context.Context, cmd *cobra.Command, args []string) error {
	logger, err := micrologger.New(micrologger.Config{})
	if err != nil {
		return microerror.Mask(err)
	}

	if existingCluster {
		if k8sApiUrl == "" {
			return microerror.Maskf(invalidConfigError, "flag --k8s-api-url must not be empty")
		}

		if k8sCert == "" {
			return microerror.Maskf(invalidConfigError, "flag --k8s-cert must not be empty")
		}

		if k8sCertCA == "" {
			return microerror.Maskf(invalidConfigError, "flag --k8s-cert-ca must not be empty")
		}

		if k8sCertKey == "" {
			return microerror.Maskf(invalidConfigError, "flag --k8s-cert-key  must not be empty")
		}
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

			Dir:             setupCmdTestDir,
			ExistingCluster: existingCluster,
			ImageTag:        e2eHarnessTag,
			RemoteCluster:   remoteCluster,
		}

		d = docker.New(c)
	}

	pa := patterns.New(logger)
	w := wait.New(logger, d, pa)

	pCfg := &project.Config{
		Name:       projectName,
		K8sContext: k8sContext,
		Tag:        projectTag,
	}

	fs := afero.NewOsFs()
	pDeps := &project.Dependencies{
		Logger: logger,
		Runner: d,
		Wait:   w,
		Fs:     fs,
	}
	p := project.New(pDeps, pCfg)
	hCfg := harness.Config{
		ExistingCluster: existingCluster,
		RemoteCluster:   remoteCluster,
	}
	h := harness.New(logger, fs, hCfg)

	clusterCfg := cluster.Config{
		Logger:          logger,
		Fs:              fs,
		ExistingCluster: existingCluster,
		K8sApiUrl:       k8sApiUrl,
		K8sCert:         k8sCert,
		K8sCertCA:       k8sCertCA,
		K8sCertKey:      k8sCertKey,
		RemoteCluster:   remoteCluster,
		Runner:          d,
	}

	c := cluster.New(clusterCfg)

	// tasks to run
	bundle := []tasks.Task{
		h.Init,
		h.WriteConfig,
		c.Create,
	}
	if !existingCluster {
		bundle = append(bundle, p.CommonSetupSteps)
	}

	return microerror.Mask(tasks.Run(ctx, bundle))
}
