package template

import (
	_ "embed"
	"encoding/json"
	"strings"

	"github.com/giantswarm/microerror"
)

//go:embed main.json
var template string

// GetARMTemplate returns the ARM template reading a json file locally using go embed.
func GetARMTemplate() (map[string]interface{}, error) {
	contents := make(map[string]interface{})

	d := json.NewDecoder(strings.NewReader(template))
	if err := d.Decode(&contents); err != nil {
		return contents, microerror.Mask(err)
	}
	return contents, nil
}
