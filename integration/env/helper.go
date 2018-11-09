package env

import (
	"fmt"
	"os"
)

func getEnv(name string) string {
	return os.Getenv(name)
}

func mustGetEnv(name string) string {
	v := getEnv(name)
	if v == "" {
		panic(fmt.Sprintf("env var %#q must not be empty", name))
	}
}
