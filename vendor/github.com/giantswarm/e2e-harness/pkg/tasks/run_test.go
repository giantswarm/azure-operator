package tasks_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/spf13/afero"

	"github.com/giantswarm/e2e-harness/pkg/tasks"
)

var (
	files   = []string{"/task1", "/task2"}
	taskErr = func(ctx context.Context) error {
		return fmt.Errorf("my-error")
	}
)

func getTaskFunc(filename string, fs afero.Fs) tasks.Task {
	return func(ctx context.Context) error {
		if err := afero.WriteFile(fs, filename, []byte("test!"), 0644); err != nil {
			return err
		}
		return nil
	}
}

func TestRunNoError(t *testing.T) {
	ctx := context.Background()

	fs := new(afero.MemMapFs)

	bundle := []tasks.Task{getTaskFunc(files[0], fs), getTaskFunc(files[1], fs)}

	err := tasks.Run(ctx, bundle)
	if err != nil {
		t.Errorf("unexpected error %s", err)
	}

	for _, file := range files {
		e, err := afero.Exists(fs, file)
		if err != nil {
			t.Errorf("unexpected error %s", err)
		}
		if !e {
			t.Errorf("expected file %s to exists", file)
		}
	}
}

func TestRunError(t *testing.T) {
	ctx := context.Background()

	fs := new(afero.MemMapFs)

	var bundle []tasks.Task
	bundle = append(bundle, getTaskFunc(files[0], fs))
	bundle = append(bundle, taskErr)
	bundle = append(bundle, getTaskFunc(files[1], fs))

	err := tasks.Run(ctx, bundle)
	if err == nil {
		t.Error("expected error didn't happen")
	}
	if err.Error() != "my-error" {
		t.Error("expected error didn't happen")
	}

	e, err := afero.Exists(fs, files[0])
	if err != nil {
		t.Errorf("unexpected error %s", err)
	}
	if !e {
		t.Errorf("expected file %s to exists", files[0])
	}

}
