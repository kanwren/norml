package norml

import (
	_ "embed"
	"fmt"
	"runtime"
	"strings"
)

const AppName = "norml"

// go build -ldflags "-X github.com/kanwren/norml.AppVersionMetadata=$(date -u +%s)"
var AppVersionMetadata string

//go:embed version
var AppVersion string

func Version() string {
	suffix := ""
	if AppVersionMetadata != "" {
		suffix = "-" + AppVersionMetadata
	}
	v := strings.TrimSpace(AppVersion)
	return fmt.Sprintf("%s %s%s (Go runtime %s)", AppName, v, suffix, runtime.Version())
}
