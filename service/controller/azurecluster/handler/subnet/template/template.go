package subnet

import (
	_ "embed"
	"encoding/json"
	"strings"

	"github.com/giantswarm/microerror"
)

// GetARMTemplate returns the ARM template reading a json file locally using go embed.
func GetARMTemplate() (map[string]interface{}, error) {
	contents := make(map[string]interface{})

	// go:embed main.json
	var template string

	d := json.NewDecoder(strings.NewReader(template))
	if err := d.Decode(&contents); err != nil {
		return contents, microerror.Mask(err)
	}
	return contents, nil
}
