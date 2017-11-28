package cloudconfig

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"

	"github.com/giantswarm/azuretpr"
	k8scloudconfig "github.com/giantswarm/k8scloudconfig/v_0_1_0"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/viper"

	"github.com/giantswarm/azure-operator/flag"
)

// Config represents the configuration used to create a cloudconfig service.
type Config struct {
	// Dependencies.

	Logger micrologger.Logger

	// Settings.

	Flag  *flag.Flag
	Viper *viper.Viper
}

// DefaultConfig provides a default configuration to create a new cloudconfig service
// by best effort.
func DefaultConfig() Config {
	return Config{
		// Dependencies.
		Logger: nil,

		// Settings.
		Flag:  nil,
		Viper: nil,
	}
}

// CloudConfig implements the cloudconfig service interface.
type CloudConfig struct {
	// Dependencies.
	logger micrologger.Logger
}

// New creates a new configured cloudconfig service.
func New(config Config) (*CloudConfig, error) {
	// Dependencies.
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "logger must not be empty")
	}

	// Settings.
	if config.Flag == nil {
		return nil, microerror.Maskf(invalidConfigError, "flag must not be empty")
	}
	if config.Viper == nil {
		return nil, microerror.Maskf(invalidConfigError, "viper must not be empty")
	}

	newService := &CloudConfig{
		// Dependencies.
		logger: config.Logger,
	}

	return newService, nil
}

// NewMasterCloudConfig generates a new master cloudconfig and returns it as a
// base64 encoded string.
func (c CloudConfig) NewMasterCloudConfig(customObject azuretpr.CustomObject) (string, error) {
	params := k8scloudconfig.Params{
		Cluster: customObject.Spec.Cluster,
		Extension: &MasterExtension{
			CloudConfigExtension{
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
		Extension: &WorkerExtension{
			CloudConfigExtension{
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
