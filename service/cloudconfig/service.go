package cloudconfig

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azuretpr"
	k8scloudconfig "github.com/giantswarm/k8scloudconfig/v_0_1_0"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
)

// Config represents the configuration used to create a cloudconfig service.
type Config struct {
	// Dependencies.

	Logger micrologger.Logger

	// Settings.

	// TODO(pk) remove as soon as we sort calico in Azure provider.
	AzureConfig client.AzureConfig
}

// DefaultConfig provides a default configuration to create a new cloudconfig service
// by best effort.
func DefaultConfig() Config {
	return Config{
		// Dependencies.
		Logger: nil,

		// Settings.
		AzureConfig: client.DefaultAzureConfig(),
	}
}

// CloudConfig implements the cloudconfig service interface.
type CloudConfig struct {
	// Dependencies.
	logger micrologger.Logger

	// Settings.
	azureConfig client.AzureConfig
}

// New creates a new configured cloudconfig service.
func New(config Config) (*CloudConfig, error) {
	// Dependencies.
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.Logger must not be empty")
	}

	// Settings.
	if err := config.AzureConfig.Validate(); err != nil {
		return nil, microerror.Maskf(invalidConfigError, "config.AzureConfig.%s", err)
	}

	newService := &CloudConfig{
		// Dependencies.
		logger: config.Logger,

		// Settings.
		azureConfig: config.AzureConfig,
	}

	return newService, nil
}

// NewMasterCloudConfig generates a new master cloudconfig and returns it as a
// base64 encoded string.
func (c CloudConfig) NewMasterCloudConfig(customObject azuretpr.CustomObject) (string, error) {
	params := k8scloudconfig.Params{
		Cluster: customObject.Spec.Cluster,
		Extension: &masterExtension{
			cloudConfigExtension{
				AzureConfig:  c.azureConfig,
				CustomObject: customObject,
			},
		},
	}

	return c.newCloudConfig(k8scloudconfig.MasterTemplate, params)
}

// NewWorkerCloudConfig generates a new worker cloudconfig and returns it as a
// base64 encoded string.
func (c CloudConfig) NewWorkerCloudConfig(customObject azuretpr.CustomObject) (string, error) {
	params := k8scloudconfig.Params{
		Cluster: customObject.Spec.Cluster,
		Extension: &workerExtension{
			cloudConfigExtension{
				AzureConfig:  c.azureConfig,
				CustomObject: customObject,
			},
		},
	}

	return c.newCloudConfig(k8scloudconfig.WorkerTemplate, params)
}

func (c CloudConfig) newCloudConfig(template string, params k8scloudconfig.Params) (string, error) {
	cloudConfig, err := k8scloudconfig.NewCloudConfig(template, params)
	if err != nil {
		return "", microerror.Mask(err)
	}
	err = cloudConfig.ExecuteTemplate()
	if err != nil {
		return "", microerror.Mask(err)
	}

	compressed, err := gzipBase64(cloudConfig.String())
	if err != nil {
		return "", microerror.Maskf(err, "compressing cloud-config")
	}

	// cloud-config is compressed so we fit the tight 65.5kB limit of ARM
	// template parameter size.
	customData := fmt.Sprintf(`#!/bin/bash
D="/root/cloudinit"
mkdir -m 700 -p ${D}
echo -n "%s" | base64 -d | gzip -d -c > ${D}/cc
coreos-cloudinit --from-file=${D}/cc`, compressed)

	// TODO use base64() in ARM template
	customData = base64.StdEncoding.EncodeToString([]byte(customData))
	return customData, nil
}

func gzipBase64(s string) (string, error) {
	buf := new(bytes.Buffer)

	w, err := gzip.NewWriterLevel(buf, gzip.BestCompression)
	if err != nil {
		return "", microerror.Maskf(err, "creating gzip stream")
	}
	_, err = io.WriteString(w, s)
	if err != nil {
		return "", microerror.Maskf(err, "writing to gzip stream")
	}
	err = w.Close()
	if err != nil {
		return "", microerror.Maskf(err, "closing gzip stream")
	}

	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}
