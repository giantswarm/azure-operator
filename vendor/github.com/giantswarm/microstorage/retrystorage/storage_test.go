package retrystorage

import (
	"testing"

	"github.com/giantswarm/micrologger/microloggertest"
	"github.com/giantswarm/microstorage/memory"
	"github.com/giantswarm/microstorage/storagetest"
)

func TestRetryStorage(t *testing.T) {
	underlying, err := memory.New(memory.DefaultConfig())
	if err != nil {
		t.Fatalf("unexpected error %#v", err)
	}

	c := Config{
		Logger:     microloggertest.New(),
		Underlying: underlying,
	}

	storage, err := New(c)
	if err != nil {
		t.Fatalf("unexpected error %#v", err)
	}

	storagetest.Test(t, storage)
}
