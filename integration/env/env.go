package env

import (
	"crypto/sha1"
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	EnvClusterName   = "CLUSTER_NAME"
	EnvCircleSHA     = "CIRCLE_SHA1"
	EnvCircleJobName = "CIRCLE_JOB"
	EnvCircleJobID   = "CIRCLE_WORKFLOW_JOB_ID"
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
}

func ClusterName() string {
	var parts []string

	parts = append(parts, "ci")
	parts = append(parts, CircleJobName())
	parts = append(parts, CircleJobID()[0:5])
	parts = append(parts, CircleSHA()[0:5])

	return strings.Join(parts, "-")
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
