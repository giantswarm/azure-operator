package tasks

import (
	"context"
)

// Task represent a generic step in a pipeline.
type Task func(ctx context.Context) error
