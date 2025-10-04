package cmdutil

import (
	"runtime/debug"
	"strings"

	versioninfo "taxowalk"
)

func ResolveVersion(override string) string {
	if v := strings.TrimSpace(override); v != "" && v != "dev" {
		return v
	}
	if v := versioninfo.Value(); v != "" {
		return v
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		if v := strings.TrimSpace(info.Main.Version); v != "" && v != "(devel)" {
			return v
		}
	}
	return "dev"
}
