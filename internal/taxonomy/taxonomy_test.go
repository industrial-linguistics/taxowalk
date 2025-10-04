package taxonomy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
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

func TestFetchCachesHTTPResponse(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	var hits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&hits, 1) > 1 {
			t.Fatalf("expected cached fetch to avoid additional HTTP requests")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"version":"test","verticals":[{"name":"Root","categories":[{"id":"1","level":1,"name":"Root","full_name":"Root","children":[]}]}]}`))
	}))
	t.Cleanup(server.Close)

	tax, err := Fetch(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("Fetch returned error: %v", err)
	}
	if tax.Version != "test" {
		t.Fatalf("unexpected version: %s", tax.Version)
	}

	tax, err = Fetch(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("Fetch returned error on cached read: %v", err)
	}
	if tax.Version != "test" {
		t.Fatalf("unexpected version on cached read: %s", tax.Version)
	}
	if atomic.LoadInt32(&hits) != 1 {
		t.Fatalf("expected exactly one HTTP request, got %d", hits)
	}
}

func TestFetchCacheExpires(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)
	hit := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"version":"test","verticals":[{"name":"Root","categories":[{"id":"1","level":1,"name":"Root","full_name":"Root","children":[]}]}]}`))
	}))
	defer server.Close()

	if _, err := Fetch(context.Background(), server.URL); err != nil {
		t.Fatalf("Fetch returned error: %v", err)
	}
	if hit != 1 {
		t.Fatalf("expected 1 HTTP hit, got %d", hit)
	}

	path, err := cacheFilePath(server.URL)
	if err != nil {
		t.Fatalf("cacheFilePath returned error: %v", err)
	}
	expired := time.Now().Add(-2 * cacheMaxAge)
	if err := os.Chtimes(path, expired, expired); err != nil {
		t.Fatalf("failed to update cache timestamp: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	server.Close()

	if _, err := Fetch(ctx, server.URL); err == nil {
		t.Fatalf("expected error when cache expired and server unavailable")
	}
}
