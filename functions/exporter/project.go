package exporter

import (
	"os"
)

var KnownProjectIDEnvs = []string{
	"CLOUDSDK_CORE_PROJECT",
	"GOOGLE_CLOUD_PROJECT",
}

func getProjectID(defaultProjectID string) string {
	for _, env := range KnownProjectIDEnvs {
		value, ok := os.LookupEnv(env)
		if ok {
			return value
		}
	}
	return defaultProjectID
}
