package versioninfo

import (
	_ "embed"
	"strings"
)

//go:embed VERSION
var rawVersion string

// Value returns the embedded version string trimmed of whitespace.
func Value() string {
	return strings.TrimSpace(rawVersion)
}
