package templates_test

import (
	"testing"

	"github.com/giantswarm/azure-operator/v8/service/controller/templates"
)

func TestRender(t *testing.T) {
	t.Parallel()
	tpl := "some string <{{.Value}}> another string"
	d := struct {
		Value string
	}{"myvalue"}
	expected := "some string <myvalue> another string"

	actual, err := templates.Render([]string{tpl}, d)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	if actual != expected {
		t.Errorf("unexpected output, want %q, got %q", expected, actual)
	}
}
