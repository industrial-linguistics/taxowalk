package taxonomy

import (
	"context"
	"path/filepath"
	"testing"
)

func TestFetchFromFile(t *testing.T) {
	path := filepath.Join("testdata", "sample.json")
	tax, err := Fetch(context.Background(), path)
	if err != nil {
		t.Fatalf("Fetch returned error: %v", err)
	}
	if tax.Version != "test" {
		t.Fatalf("unexpected version: %s", tax.Version)
	}
	if len(tax.Roots) != 1 {
		t.Fatalf("expected 1 root, got %d", len(tax.Roots))
	}
	root := tax.Roots[0]
	if root.Name != "Root" {
		t.Fatalf("unexpected root name: %s", root.Name)
	}
	if root.FindChildByName("Child") == nil {
		t.Fatalf("child not found")
	}
}
