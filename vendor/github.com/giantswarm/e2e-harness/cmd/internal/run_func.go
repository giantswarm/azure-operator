package internal

import (
	"context"
	"fmt"
	"os"

	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"
)

// NewRunFunc creates a wrapper for running functiosn in cobra comands. It sets
// up a context and logger and ensures good error logging.
func NewRunFunc(fn func(ctx context.Context, cmd *cobra.Command, args []string) error) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		ctx := context.Background()

		logger, err := micrologger.New(micrologger.Config{})
		if err != nil {
			panic(fmt.Sprintf("%#v", err))
		}

		err = fn(ctx, cmd, args)
		if err != nil {
			logger.LogCtx(ctx, "level", "error", "message", "exiting with status 1 due to error", "stack", fmt.Sprintf("%#v", err))
			os.Exit(1)
		}
	}
}
