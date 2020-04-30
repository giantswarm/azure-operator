package template

import (
	"encoding/json"

	"github.com/giantswarm/microerror"
	"github.com/markbates/pkger"
)

// GetARMTemplate returns the ARM template reading a json file locally using pkger.
func GetARMTemplate() (map[string]interface{}, error) {
	contents := make(map[string]interface{})

	f, err := pkger.Open("/service/controller/resource/masters/template/main.json")
	if err != nil {
		return contents, microerror.Mask(err)
	}
	defer f.Close()

	d := json.NewDecoder(f)
	if err := d.Decode(&contents); err != nil {
		return contents, microerror.Mask(err)
	}
	return contents, microerror.Mask(err)
}
