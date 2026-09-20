package updatecheck

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

func TestCheckUsesSemVerPrereleasePrecedenceAndIgnoresBuildMetadata(t *testing.T) {
	for _, test := range []struct {
		current, latest string
		want            string
	}{
		{current: "v1.2.0-beta", latest: "v1.2.0", want: "v1.2.0"},
		{current: "v1.2.0", latest: "v1.2.0+build.7", want: ""},
		{current: "v1.2.0-alpha.1", latest: "v1.2.0-alpha.2", want: "v1.2.0-alpha.2"},
		{current: "v1.2.0-alpha.2", latest: "v1.2.0-alpha.10", want: "v1.2.0-alpha.10"},
	} {
		t.Run(test.current+"_"+test.latest, func(t *testing.T) {
			if got := newer(test.current, test.latest); got != test.want {
				t.Fatalf("newer(%q, %q) = %q, want %q", test.current, test.latest, got, test.want)
			}
		})
	}
}

func TestParseVersionRejectsMalformedSemVerIdentifiers(t *testing.T) {
	for _, version := range []string{
		"v1.2.3-",
		"v1.2.3+",
		"v1.2.3-rc..1",
		"v1.2.3+build..1",
		"v1.2.3-01",
		"v1.2.3-rc.01",
	} {
		t.Run(version, func(t *testing.T) {
			if _, err := parseVersion(version); err == nil {
				t.Fatalf("parseVersion(%q) accepted malformed SemVer", version)
			}
		})
	}
}

func TestCheckRepairsExistingCachePermissions(t *testing.T) {
	state := t.TempDir()
	path := filepath.Join(state, cacheFileName)
	data, err := json.Marshal(cache{CheckedAt: now().UTC(), LatestVersion: "v1.3.0"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Check(context.Background(), state, "v1.2.1"); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("cache permissions = %v, %v", info, err)
	}
}

func TestCheckCachesFailedAttemptForTwentyFourHours(t *testing.T) {
	state := t.TempDir()
	requests := 0
	oldEndpoint, oldNow, oldHTTPClient := endpoint, now, httpClient
	endpoint = "https://updates.invalid/latest"
	httpClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		if requests == 1 {
			return &http.Response{StatusCode: http.StatusServiceUnavailable, Body: io.NopCloser(strings.NewReader("temporarily unavailable")), Header: make(http.Header)}, nil
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"tag_name":"v1.3.0","draft":false,"prerelease":false}`)), Header: make(http.Header)}, nil
	})}
	clock := time.Date(2026, 9, 20, 1, 0, 0, 0, time.UTC)
	now = func() time.Time { return clock }
	t.Cleanup(func() { endpoint, now, httpClient = oldEndpoint, oldNow, oldHTTPClient })

	if got, err := Check(context.Background(), state, "v1.2.1"); err == nil || got != "" {
		t.Fatalf("failed check = %q, %v", got, err)
	}
	clock = clock.Add(time.Hour)
	if got, err := Check(context.Background(), state, "v1.2.1"); err != nil || got != "" {
		t.Fatalf("cached failed check = %q, %v", got, err)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
}

func TestCheckDoesNotRequestWhenAttemptCacheCannotBeWritten(t *testing.T) {
	parent := t.TempDir()
	state := filepath.Join(parent, "state-file")
	if err := os.WriteFile(state, []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	requests := 0
	oldEndpoint, oldHTTPClient := endpoint, httpClient
	endpoint = "https://updates.invalid/latest"
	httpClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"tag_name":"v1.3.0","draft":false,"prerelease":false}`)),
			Header:     make(http.Header),
		}, nil
	})}
	t.Cleanup(func() { endpoint, httpClient = oldEndpoint, oldHTTPClient })

	if got, err := Check(context.Background(), state, "v1.2.1"); err == nil || got != "" {
		t.Fatalf("uncacheable check = %q, %v", got, err)
	}
	if requests != 0 {
		t.Fatalf("requests = %d, want 0", requests)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }
