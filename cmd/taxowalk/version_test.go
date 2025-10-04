package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	versioninfo "taxowalk"
)

func TestEmbeddedVersionMatchesRoot(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("unable to determine caller information")
	}
	rootVersionPath := filepath.Join(filepath.Dir(filename), "..", "..", "VERSION")
	data, err := os.ReadFile(rootVersionPath)
	if err != nil {
		t.Fatalf("failed to read root VERSION file: %v", err)
	}
	rootVersion := strings.TrimSpace(string(data))
	embedded := versioninfo.Value()
	if rootVersion == "" {
		t.Fatal("root VERSION file is empty")
	}
	if embedded == "" {
		t.Fatal("embedded version is empty")
	}
	if rootVersion != embedded {
		t.Fatalf("embedded version %q does not match root VERSION %q", embedded, rootVersion)
	}
}
