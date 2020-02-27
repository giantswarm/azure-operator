package project

var (
	bundleVersion        = "3.0.0-dev"
	description   string = "The azure-operator manages Kubernetes clusters on Azure."
	gitSHA               = "n/a"
	name          string = "azure-operator"
	source        string = "https://github.com/giantswarm/azure-operator"
	version              = "n/a"
)

func BundleVersion() string {
	return bundleVersion
}

func Description() string {
	return description
}

func GitSHA() string {
	return gitSHA
}

func Name() string {
	return name
}

func Source() string {
	return source
}

func Version() string {
	return version
}
