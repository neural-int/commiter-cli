package updatecheck

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCheckFetchesStableReleaseAndCaches(t *testing.T) {
	state := t.TempDir()
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Path != "/latest" || r.Header.Get("User-Agent") == "" {
			t.Fatalf("unexpected request: %s %q", r.URL.Path, r.Header.Get("User-Agent"))
		}
		_, _ = w.Write([]byte(`{"tag_name":"v1.3.0","draft":false,"prerelease":false}`))
	}))
	defer server.Close()
	oldEndpoint, oldNow := endpoint, now
	endpoint = server.URL + "/latest"
	clock := time.Date(2026, 9, 20, 1, 0, 0, 0, time.UTC)
	now = func() time.Time { return clock }
	t.Cleanup(func() { endpoint, now = oldEndpoint, oldNow })

	got, err := Check(context.Background(), state, "v1.2.1")
	if err != nil || got != "v1.3.0" {
		t.Fatalf("first check = %q, %v", got, err)
	}
	clock = clock.Add(time.Hour)
	got, err = Check(context.Background(), state, "v1.2.1")
	if err != nil || got != "v1.3.0" || requests != 1 {
		t.Fatalf("cached check = %q, %v, requests=%d", got, err, requests)
	}
	info, err := os.Stat(filepath.Join(state, cacheFileName))
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("cache permissions = %v, %v", info, err)
	}
}

func TestCheckIgnoresCurrentAndUnstableReleases(t *testing.T) {
	state := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"tag_name":"v2.0.0-rc1","draft":false,"prerelease":true}`))
	}))
	defer server.Close()
	oldEndpoint := endpoint
	endpoint = server.URL
	t.Cleanup(func() { endpoint = oldEndpoint })
	if got, err := Check(context.Background(), state, "v2.0.0"); err == nil || got != "" {
		t.Fatalf("unstable release = %q, %v", got, err)
	}

	data, _ := json.Marshal(cache{CheckedAt: now().UTC(), LatestVersion: "v1.2.1"})
	if err := os.WriteFile(filepath.Join(state, cacheFileName), data, 0o600); err != nil {
		t.Fatal(err)
	}
	if got, err := Check(context.Background(), state, "v1.2.1"); err != nil || got != "" {
		t.Fatalf("same version = %q, %v", got, err)
	}
}

func TestCheckSkipsDevelopmentVersion(t *testing.T) {
	if got, err := Check(context.Background(), t.TempDir(), "dev"); err != nil || got != "" {
		t.Fatalf("development version = %q, %v", got, err)
	}
}
