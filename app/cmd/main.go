package main

import (
	"github.com/mitchs-dev/simplQL/pkg/api/requests"
	"github.com/mitchs-dev/simplQL/pkg/configurationAndInitalization/globals"
	"github.com/mitchs-dev/simplQL/pkg/configurationAndInitalization/initalization"
	"github.com/mitchs-dev/simplQL/pkg/configurationAndInitalization/version"
	log "github.com/sirupsen/logrus"
)

func init() {
	initalization.Run()
}

func main() {
	log.Info(globals.ApplicationName + " (v" + version.SymanticString() + ")")
	requests.Handler()
}
