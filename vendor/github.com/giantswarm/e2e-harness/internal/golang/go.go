package golang

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/giantswarm/e2e-harness/internal/docker"
	"github.com/giantswarm/microerror"
)

// Go runs a go command for a project in a given working directory. Example
// call could look like this:
//
//	Go(ctx, "build", "-o", "e2e-harness", ".")
//
func Go(ctx context.Context, args ...string) error {
	var err error

	// Check as much as possible before executing docker image. After that
	// error messages are worth nothing and we need to check output.

	{
		if len(args) == 0 {
			return microerror.Maskf(executionFailedError, "args for executing go must be set")
		}
		if args[0] == "go" {
			return microerror.Maskf(executionFailedError, "args for executing go not contain %#q but got %#v", "go", args)
		}
	}

	var hostWdir string
	{
		hostWdir, err = os.Getwd()
		if err != nil {
			return microerror.Mask(err)
		}
	}

	var hostGoPath string
	{
		v := os.Getenv(envVarGoPath)
		if v == "" {
			return microerror.Maskf(executionFailedError, "environment variable %#q must not be empty", envVarGoPath)
		}

		split := strings.Split(v, ":")

		for _, s := range split {
			if strings.HasPrefix(hostWdir, s) {
				hostGoPath = s
				break
			}
		}

		if hostGoPath == "" {
			return microerror.Maskf(executionFailedError, "expected current working directory %#q to be in GOPATH %#q", hostWdir, hostGoPath)
		}
	}

	var containerGoPath string
	{
		containerGoPath = "/go"
	}

	var containerWdir string
	{
		relWdir := strings.TrimPrefix(hostWdir, hostGoPath)

		containerWdir = filepath.Join(containerGoPath, relWdir)
	}

	c := docker.RunConfig{
		Volumes: []string{
			fmt.Sprintf("%s:%s", hostGoPath, containerGoPath),
		},
		Env: []string{
			"CGO_ENABLED=" + cgoEnabled,
			"GOARCH=" + goArch,
			"GOCACHE=" + filepath.Join(containerGoPath, "cache"),
			"GOOS=" + goOS,
			"GOPATH=" + containerGoPath,
		},
		WorkingDirectory: containerWdir,
		Image:            dockerImage,
		Args:             append([]string{"go"}, args...),
	}

	err = docker.Run(ctx, c)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
