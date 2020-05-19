package project

import (
	"fmt"
)

var (
	description = "The azure-operator manages Kubernetes clusters on Azure."
	gitSHA      = "n/a"
	name        = "azure-operator"
	source      = "https://github.com/giantswarm/azure-operator"
	version     = "4.1.0"
	wipSuffix   = "-dev"
)

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
	return fmt.Sprintf("%s%s", version, wipSuffix)
}
