package apprclienttest

import (
	"testing"
)

func Test_New(t *testing.T) {
	// Test that New doesn't panic and apprclient.Interface is implemented.
	New(Config{})
}
