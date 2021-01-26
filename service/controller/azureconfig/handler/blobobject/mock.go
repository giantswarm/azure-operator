package blobobject

import (
	"context"

	"github.com/giantswarm/azure-operator/v5/service/controller/cloudconfig"
	"github.com/giantswarm/azure-operator/v5/service/controller/encrypter"
)

type CloudConfigMock struct {
	template string
}

func (c *CloudConfigMock) NewMasterTemplate(ctx context.Context, data cloudconfig.IgnitionTemplateData, encrypter encrypter.Interface) (string, error) {
	return c.template, nil
}

func (c *CloudConfigMock) NewWorkerTemplate(ctx context.Context, data cloudconfig.IgnitionTemplateData, encrypter encrypter.Interface) (string, error) {
	return c.template, nil
}
