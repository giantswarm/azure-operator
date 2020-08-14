package env

import (
	"fmt"
	"os"
	"strconv"
)

const (
	EnvVarIgnitionDebugEnabled    = "IGNITION_DEBUG_ENABLED"
	EnvVarIgnitionDebugLogsPrefix = "IGNITION_DEBUG_LOGS_PREFIX"
	EnvVarIgnitionDebugLogsToken  = "IGNITION_DEBUG_LOGS_TOKEN" // #nosec
)

var (
	ignitionDebugEnabled bool
	ignitionLogsPrefix   string
	ignitionLogsToken    string
)

func init() {
	var err error

	ignitionDebugEnabled, err = strconv.ParseBool(os.Getenv(EnvVarIgnitionDebugEnabled))
	if err != nil {
		ignitionDebugEnabled = false
		fmt.Printf("failed to parse `%s` as boolean value, setting to false\n", EnvVarIgnitionDebugEnabled)
	}

	if ignitionDebugEnabled {
		ignitionLogsPrefix = os.Getenv(EnvVarIgnitionDebugLogsPrefix)
		if ignitionLogsPrefix == "" {
			panic(fmt.Sprintf("env var '%s' must not be empty", EnvVarIgnitionDebugLogsPrefix))
		}

		ignitionLogsToken = os.Getenv(EnvVarIgnitionDebugLogsToken)
		if ignitionLogsToken == "" {
			panic(fmt.Sprintf("env var '%s' must not be empty", EnvVarIgnitionDebugLogsToken))
		}
	}
}

func IgnitionDebugEnabled() bool {
	return ignitionDebugEnabled
}

func IgnitionDebugLogsPrefix() string {
	return ignitionLogsPrefix
}

func IgnitionDebugLogsToken() string {
	return ignitionLogsToken
}
