package docker

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/e2e-harness/pkg/harness"
)

type Config struct {
	Logger micrologger.Logger

	Dir             string
	ExistingCluster bool
	ImageTag        string
	RemoteCluster   bool
}

type Docker struct {
	logger micrologger.Logger

	dir             string
	existingCluster bool
	imageTag        string
	remoteCluster   bool
}

func New(config Config) *Docker {
	d := &Docker{
		logger: config.Logger,

		dir:             config.Dir,
		existingCluster: config.ExistingCluster,
		imageTag:        config.ImageTag,
		remoteCluster:   config.RemoteCluster,
	}

	return d
}

// RunPortForward executes a command in the e2e-harness container after
// setting up the port forwarding to the remote cluster, this command
// is meant to be used after that cluster has been initialized
func (d *Docker) RunPortForward(out io.Writer, command string, env ...string) error {
	if !d.remoteCluster || d.existingCluster {
		// no need to port forward in local clusters or on existing cluster
		return d.Run(out, command, env...)
	}

	args := append([]string{
		"quay.io/giantswarm/e2e-harness:" + d.imageTag},
		"-c", fmt.Sprintf("shipyard -action=forward-port && %s", command))

	return d.baseRun(out, "/bin/bash", args, env...)
}

// Run executes a command in the e2e-harness container.
func (d *Docker) Run(out io.Writer, command string, env ...string) error {
	args := append([]string{
		"quay.io/giantswarm/e2e-harness:" + d.imageTag},
		"-c", command)

	return d.baseRun(out, "/bin/bash", args, env...)
}

func (d *Docker) baseRun(out io.Writer, entrypoint string, args []string, env ...string) error {
	baseDir, err := harness.BaseDir()
	if err != nil {
		return microerror.Mask(err)
	}

	e2eDir := filepath.Join(filepath.Dir(baseDir), d.dir)
	baseArgs := []string{
		"run",
		"-v", fmt.Sprintf("%s:%s:z", filepath.Join(baseDir, "workdir"), "/workdir"),
		"-v", fmt.Sprintf("%s:/e2e:z", e2eDir),
		"-e", fmt.Sprintf("AWS_ACCESS_KEY_ID=%s", os.Getenv("AWS_ACCESS_KEY_ID")),
		"-e", fmt.Sprintf("AWS_SECRET_ACCESS_KEY=%s", os.Getenv("AWS_SECRET_ACCESS_KEY")),
		"-e", "KUBECONFIG=" + harness.DefaultKubeConfig,
		"--dns", "1.1.1.1",
		"--entrypoint", entrypoint,
		"--cap-add", "NET_ADMIN",
	}

	// add-host entries, the ADD_HOSTS env var will look like:
	// host1:ip1,host2:ip2...
	addHostEntries := strings.Split(os.Getenv("ADD_HOSTS"), ",")
	for _, entry := range addHostEntries {
		if entry != "" {
			baseArgs = append(baseArgs, "--add-host", entry)
		}
	}

	// add environment variables
	for _, e := range env {
		sEnv := os.ExpandEnv(e)
		baseArgs = append(baseArgs, "-e", sEnv)
	}

	if !d.remoteCluster {
		// accessing to local cluster requires using the host network
		baseArgs = append(baseArgs, "--network", "host")
	}

	baseArgs = append(baseArgs, args...)

	cmd := exec.Command("docker", baseArgs...)
	cmd.Stdout = out
	cmd.Stderr = out

	return cmd.Run()
}

func (d *Docker) Build(out io.Writer, image, path, tag string, env []string) error {
	dir, err := os.Getwd()
	if err != nil {
		return microerror.Mask(err)
	}
	dockerFilePath := filepath.Join(dir, "Dockerfile")
	_, err = os.Stat(dockerFilePath)
	if os.IsNotExist(err) {
		d.logger.Log("function", "Build", "level", "info", "message", "no Dockerfile, skipping image build")
		return nil
	}

	baseArgs := []string{
		"build",
		"--no-cache",
		"-t", fmt.Sprintf("%s:%s", image, tag),
		".",
	}
	cmd := exec.Command("docker", baseArgs...)
	cmd.Stdout = out
	cmd.Stderr = out
	cmd.Dir = path
	cmd.Env = env

	return cmd.Run()
}
