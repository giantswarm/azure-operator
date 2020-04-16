package appcatalog

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"

	"github.com/ghodss/yaml"
	"github.com/giantswarm/microerror"
)

// GetLatestChart returns the latest chart tarball file for the specified storage URL and app
// and returns notFoundError when it can't find a specified app.
func GetLatestChart(ctx context.Context, storageURL, app string) (string, error) {
	index, err := getIndex(storageURL)
	if err != nil {
		return "", microerror.Mask(err)
	}

	var downloadURL string
	{
		entry, ok := index.Entries[app]
		if !ok {
			return "", microerror.Maskf(notFoundError, "no app %#q in index.yaml", app)
		}
		downloadURL = entry[0].Urls[0]
	}

	return downloadURL, nil
}

// GetLatestVersion returns the latest app version for the specified storage URL and app
// and returns notFoundError when it can't find a specified app.
func GetLatestVersion(ctx context.Context, storageURL, app string) (string, error) {
	index, err := getIndex(storageURL)
	if err != nil {
		return "", microerror.Mask(err)
	}

	var version string
	{
		entry, ok := index.Entries[app]
		if !ok {
			return "", microerror.Maskf(notFoundError, "no app %#q in index.yaml", app)
		}
		version = entry[0].Version
	}

	return version, nil
}

// NewTarballURL returns the chart tarball URL for the specified app and version.
func NewTarballURL(baseURL string, appName string, version string) (string, error) {
	if baseURL == "" || appName == "" || version == "" {
		return "", microerror.Maskf(executionFailedError, "baseURL %#q, appName %#q, release %#q should not be empty", baseURL, appName, version)
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", microerror.Mask(err)
	}
	u.Path = path.Join(u.Path, fmt.Sprintf("%s-%s.tgz", appName, version))
	return u.String(), nil
}

func getIndex(storageURL string) (index, error) {
	indexURL := fmt.Sprintf("%s/index.yaml", storageURL)
	resp, err := http.Get(indexURL)
	if err != nil {
		return index{}, microerror.Mask(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return index{}, microerror.Mask(err)
	}

	var i index
	err = yaml.Unmarshal(body, &i)
	if err != nil {
		return i, microerror.Mask(err)
	}

	return i, nil
}
