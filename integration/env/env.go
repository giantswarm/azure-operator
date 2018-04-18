package env

import (
	"crypto/sha1"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/giantswarm/azure-operator/integration/network"
)

const (
	EnvClusterName       = "CLUSTER_NAME"
	EnvCircleSHA         = "CIRCLE_SHA1"
	EnvCircleJobName     = "CIRCLE_JOB"
	EnvCircleJobID       = "CIRCLE_WORKFLOW_JOB_ID"
	EnvCircleBuildNumber = "CIRCLE_BUILD_NUM"

	EnvAzureCIDR             = "AZURE_CIDR"
	EnvAzureCalicoSubnetCIDR = "AZURE_CALICO_SUBNET_CIDR"
	EnvAzureMasterSubnetCIDR = "AZURE_MASTER_SUBNET_CIDR"
	EnvAzureWorkerSubnetCIDR = "AZURE_WORKER_SUBNET_CIDR"
)

var (
	circleSHA string
)

func init() {
	circleSHA = os.Getenv(EnvCircleSHA)
	if circleSHA == "" {
		panic(fmt.Sprintf("env var '%s' must not be empty", EnvCircleSHA))
	}

	clusterName := os.Getenv(EnvClusterName)
	if clusterName == "" {
		os.Setenv(EnvClusterName, ClusterName())
	}

	// azureCDIR must be provided along with other CIDRs,
	// otherwise we compute CIDRs base on EnvCircleBuildNumber value.
	azureCDIR := os.Getenv(EnvAzureCIDR)
	if azureCDIR == "" {
		buildNumber, err := strconv.ParseUint(os.Getenv(EnvCircleBuildNumber), 10, 32)
		if err != nil {
			panic(err)
		}

		cidrs, err := network.ComputeCIDR(uint(buildNumber))
		if err != nil {
			panic(err)
		}

		os.Setenv(EnvAzureCIDR, cidrs.AzureCIDR)
		os.Setenv(EnvAzureMasterSubnetCIDR, cidrs.MasterSubnetCIDR)
		os.Setenv(EnvAzureWorkerSubnetCIDR, cidrs.WorkerSubnetCIDR)
		os.Setenv(EnvAzureCalicoSubnetCIDR, cidrs.CalicoSubnetCIDR)
	} else {
		if os.Getenv(EnvAzureCalicoSubnetCIDR) == "" {
			panic(fmt.Sprintf("env var '%s' must not be empty", EnvAzureCalicoSubnetCIDR))
		}
		if os.Getenv(EnvAzureMasterSubnetCIDR) == "" {
			panic(fmt.Sprintf("env var '%s' must not be empty", EnvAzureMasterSubnetCIDR))
		}
		if os.Getenv(EnvAzureWorkerSubnetCIDR) == "" {
			panic(fmt.Sprintf("env var '%s' must not be empty", EnvAzureWorkerSubnetCIDR))
		}

	}
}

func ClusterName() string {
	var parts []string

	parts = append(parts, "ci")
	parts = append(parts, CircleJobName())
	parts = append(parts, CircleJobID()[0:5])
	parts = append(parts, CircleSHA()[0:5])

	return strings.ToLower(strings.Join(parts, "-"))
}

func CircleJobName() string {
	circleJobName := os.Getenv(EnvCircleJobName)
	if circleJobName == "" {
		circleJobName = "local"
	}

	return circleJobName
}

func CircleJobID() string {
	circleJobID := os.Getenv(EnvCircleJobID)
	if circleJobID == "" {
		// poor man's id generator
		circleJobID = fmt.Sprintf("%x", sha1.Sum([]byte(time.Now().Format(time.RFC3339Nano))))
	}

	return circleJobID
}

func CircleSHA() string {
	return circleSHA
}
