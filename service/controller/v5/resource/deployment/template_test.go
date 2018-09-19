package deployment

import (
	"strings"
	"testing"
)

func Test_TemplateURI(t *testing.T) {
	uri := templateURI("dev", "worker.json")
	euri := "https://raw.githubusercontent.com/giantswarm/azure-operator/dev/service/controller/v4/arm_templates/worker.json"

	if uri != euri {
		t.Errorf("expected '%s' got '%s'", euri, uri)
	}
}

func Test_BaseTemplateURI(t *testing.T) {
	uri := baseTemplateURI("master")
	euri := "https://raw.githubusercontent.com/giantswarm/azure-operator/master/service/controller/v4/arm_templates/"

	if uri != euri {
		t.Errorf("expected '%s', got '%s'", euri, uri)
	}

	// Additionaly make sure base URI ends with slash. This is important.
	// See main.json ARM template.
	if !strings.HasSuffix(uri, "/") {
		t.Errorf("expected '/' suffix, got '%s'", uri)
	}
}
