package mlxmodel

import (
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPlanInstallReadyUsesPinnedFilesAndNoRuntimeNetwork(t *testing.T) {
	config := []byte(`{"quantization":{"bits":4}}`)
	weights := []byte("mock model weights")
	configOID := gitBlobHash(config)
	weightHash := sha256.Sum256(weights)
	requests := 0
	spec := Spec{Repo: "owner/model", Revision: strings.Repeat("a", 40), Quantization: "4bit"}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Method != http.MethodGet || r.ContentLength > 0 || r.Header.Get("Authorization") != "" ||
			strings.Contains(r.URL.String(), "repository-diff") || strings.Contains(r.URL.String(), "prompt") {
			t.Errorf("model registry request crossed privacy boundary: method=%s url=%s", r.Method, r.URL.String())
		}
		if strings.Contains(r.URL.Path, "/tree/") {
			entries := []map[string]any{
				{"type": "file", "path": "config.json", "size": len(config), "oid": configOID},
				{"type": "file", "path": "model.safetensors", "size": len(weights), "lfs": map[string]any{"oid": hex.EncodeToString(weightHash[:])}},
			}
			_ = json.NewEncoder(w).Encode(entries)
			return
		}
		switch {
		case strings.HasSuffix(r.URL.Path, "/config.json"):
			_, _ = w.Write(config)
		case strings.HasSuffix(r.URL.Path, "/model.safetensors"):
			_, _ = w.Write(weights)
		default:
			t.Errorf("unexpected request path: %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	store := Store{Root: t.TempDir(), BaseURL: server.URL, Client: server.Client()}
	plan, err := store.Plan(context.Background(), spec)
	if err != nil || plan.Bytes != int64(len(config)+len(weights)) {
		t.Fatalf("Plan = %#v, %v", plan, err)
	}
	if requests != 1 {
		t.Fatalf("metadata requests = %d", requests)
	}
	dir, err := store.Install(context.Background(), plan)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := store.Ready(spec); err != nil || got != dir {
		t.Fatalf("Ready = %q, %v", got, err)
	}
	if requests != 3 {
		t.Fatalf("network requests before Ready = %d", requests)
	}
	if _, err := store.Ready(spec); err != nil || requests != 3 {
		t.Fatalf("Ready contacted network or failed: %v, requests=%d", err, requests)
	}
	if _, err := os.Stat(filepath.Join(dir, "model.safetensors")); err != nil {
		t.Fatal(err)
	}
}

func TestInstallRejectsDigestMismatchWithoutPublishing(t *testing.T) {
	spec := Spec{Repo: "owner/model", Revision: strings.Repeat("b", 40), Quantization: "4bit"}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("wrong"))
	}))
	defer server.Close()
	store := Store{Root: t.TempDir(), BaseURL: server.URL, Client: server.Client()}
	plan := Plan{Spec: spec, Files: []File{{Path: "config.json", Size: 5, Hash: strings.Repeat("0", 40)}}}
	if _, err := store.Install(context.Background(), plan); err == nil || !strings.Contains(err.Error(), "digest mismatch") {
		t.Fatalf("Install error = %v", err)
	}
	if _, err := store.Ready(spec); err == nil {
		t.Fatal("digest mismatch was published as ready")
	}
}

func TestPlanRejectsUnsafeRemotePath(t *testing.T) {
	spec := Spec{Repo: "owner/model", Revision: strings.Repeat("c", 40), Quantization: "4bit"}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[{"type":"file","path":"../config.json","size":1,"oid":"`+strings.Repeat("a", 40)+`"}]`)
	}))
	defer server.Close()
	store := Store{Root: t.TempDir(), BaseURL: server.URL, Client: server.Client()}
	if _, err := store.Plan(context.Background(), spec); err == nil {
		t.Fatal("unsafe remote path was accepted")
	}
}

func gitBlobHash(data []byte) string {
	hash := sha1.New()
	_, _ = fmt.Fprintf(hash, "blob %d\x00", len(data))
	_, _ = hash.Write(data)
	return hex.EncodeToString(hash.Sum(nil))
}
