package main

import (
	"os"
	"strings"

	"github.com/mitchs-dev/simplQL/pkg/api/requests"
	"github.com/mitchs-dev/simplQL/pkg/configurationAndInitialization/globals"
	"github.com/mitchs-dev/simplQL/pkg/configurationAndInitialization/initalization"
	"github.com/mitchs-dev/simplQL/pkg/configurationAndInitialization/version"
	log "github.com/sirupsen/logrus"
)

func init() {
	initalization.Run()
}

func main() {
	if strings.ToLower(os.Getenv(globals.GlobalDevelopmentBuildEnvironmentVariable)) == "true" {
		log.Warn("This is marked as a development build - Do not use in production")
		log.Info(globals.ApplicationName + " (v" + version.SymanticString() + "-dev)")
	} else {
		log.Info(globals.ApplicationName + " (v" + version.SymanticString() + ")")
	}
	requests.Handler()
}
