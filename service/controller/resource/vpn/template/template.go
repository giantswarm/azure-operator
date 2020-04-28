package template

import (
	"encoding/json"
	"io/ioutil"

	"github.com/giantswarm/microerror"
	"github.com/markbates/pkger"
)

// GetARMTemplate returns the ARM template reading a json file locally using pkger.
func GetARMTemplate() (map[string]interface{}, error) {
	contents := make(map[string]interface{})

	f, err := pkger.Open("/service/controller/resource/vpn/template/main.json")
	if err != nil {
		return contents, microerror.Mask(err)
	}
	defer f.Close()

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return contents, microerror.Mask(err)
	}

	if err := json.Unmarshal(b, &contents); err != nil {
		return nil, err
	}
	return contents, microerror.Mask(err)
}
