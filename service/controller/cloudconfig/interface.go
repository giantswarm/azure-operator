package cloudconfig

import (
	"context"

	"github.com/giantswarm/azure-operator/v8/service/controller/encrypter"
)

type Interface interface {
	NewMasterTemplate(ctx context.Context, data IgnitionTemplateData, encrypter encrypter.Interface) (string, error)
	NewWorkerTemplate(ctx context.Context, data IgnitionTemplateData, encrypter encrypter.Interface) (string, error)
}
