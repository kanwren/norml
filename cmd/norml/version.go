package main

import (
	"fmt"
	"runtime"
)

const (
	AppName         = "norml"
	AppVersionMajor = 0
	AppVersionMinor = 1
	AppVersionPatch = 0
	AppVersionBuild = ""
)

// go build -ldflags "-X main.AppVersionMetadata $(date -u +%s)"
var AppVersionMetadata string

func Version() string {
	suffix := ""

	if AppVersionBuild != "" {
		suffix += "-" + AppVersionBuild
	}

	if AppVersionMetadata != "" {
		suffix += "-" + AppVersionMetadata
	}

	return fmt.Sprintf("%s %d.%d.%d%s (Go runtime %s)", AppName, AppVersionMajor, AppVersionMinor, AppVersionPatch, suffix, runtime.Version())
}
