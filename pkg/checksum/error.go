package checksum

import (
	"github.com/giantswarm/microerror"
)

var unableToGetTemplateError = &microerror.Error{
	Kind: "unableToGetTemplate",
}

func IsUnableToGetTemplateError(err error) bool {
	return microerror.Cause(err) == unableToGetTemplateError
}

var nilTemplateLinkError = &microerror.Error{
	Kind: "nilTemplateLink",
}

func IsNilTemplateLinkError(err error) bool {
	return microerror.Cause(err) == nilTemplateLinkError
}
