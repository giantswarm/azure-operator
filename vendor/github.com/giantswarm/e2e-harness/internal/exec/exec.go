package exec

import (
	"context"
	"os"
	"os/exec"

	"github.com/giantswarm/microerror"
)

func Exec(ctx context.Context, name string, args ...string) error {
	var cmd *exec.Cmd
	{
		cmd = exec.Command(name, args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	err := cmd.Run()
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
