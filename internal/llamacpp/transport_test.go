//go:build darwin || linux

package llamacpp

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func localTestServer(t *testing.T, handler http.HandlerFunc) (string, *localTransport) {
	t.Helper()
	directory, err := os.MkdirTemp("/tmp", "c190-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(directory) })
	socket := filepath.Join(directory, "server.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(socket, 0600); err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: handler, ReadHeaderTimeout: time.Second}
	t.Cleanup(func() { _ = server.Close() })
	go server.Serve(listener)
	transport, err := newLocalTransport(socket, os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(transport.close)
	return socket, transport
}

func TestUDSTransportChecksPeerPIDBeforeSending(t *testing.T) {
	var requests atomic.Int32
	socket, transport := localTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:1")
	t.Setenv("ALL_PROXY", "http://127.0.0.1:1")
	if _, err := transport.request(context.Background(), "GET", "/health", nil); err != nil {
		t.Fatal(err)
	}
	wrong, err := newLocalTransport(socket, os.Getpid()+1)
	if err != nil {
		t.Fatal(err)
	}
	defer wrong.close()
	if _, err := wrong.request(context.Background(), "GET", "/health", nil); err == nil {
		t.Fatal("wrong peer PID accepted")
	}
	if requests.Load() != 1 {
		t.Fatal("request sent to wrong peer")
	}
	for _, path := range []string{"https://example.com/completion", "//example.com/completion", "/completion?url=external", "/slots"} {
		if _, err := transport.request(context.Background(), "POST", path, nil); err == nil {
			t.Fatal("arbitrary endpoint accepted")
		}
	}
}

func TestUDSTransportRejectsChangedSocketAndPermissions(t *testing.T) {
	socket, transport := localTestServer(t, func(http.ResponseWriter, *http.Request) { t.Error("request sent to replaced socket") })
	if err := os.Chmod(socket, 0666); err != nil {
		t.Fatal(err)
	}
	if _, err := transport.request(context.Background(), "GET", "/health", nil); err == nil {
		t.Fatal("world-accessible socket accepted")
	}
	if err := os.Chmod(socket, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(socket, socket+".old"); err != nil {
		t.Fatal(err)
	}
	replacement, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	defer replacement.Close()
	if err := os.Chmod(socket, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := transport.request(context.Background(), "GET", "/health", nil); err == nil {
		t.Fatal("replaced inode accepted")
	}
}

func TestUDSTransportEnforcesByteLimitsAndRedirectRefusal(t *testing.T) {
	for _, mode := range []string{"exact", "oversize", "header", "redirect", "error"} {
		t.Run(mode, func(t *testing.T) {
			_, transport := localTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				switch mode {
				case "redirect":
					http.Redirect(w, r, "https://example.com", http.StatusFound)
				case "header":
					w.Header().Set("X-Large", strings.Repeat("x", 33<<10))
					_, _ = w.Write([]byte("{}"))
				case "error":
					w.WriteHeader(500)
					_, _ = w.Write([]byte("PRIVATE_BODY"))
				case "exact":
					_, _ = w.Write([]byte(strings.Repeat("x", MaxResponseBytes)))
				case "oversize":
					_, _ = w.Write([]byte(strings.Repeat("x", MaxResponseBytes+1)))
				}
			})
			data, err := transport.request(context.Background(), "POST", "/completion", []byte("{}"))
			if (err == nil) != (mode == "exact") {
				t.Fatalf("mode=%s bytes=%d err=%v", mode, len(data), err)
			}
			if err != nil && strings.Contains(err.Error(), "PRIVATE_BODY") {
				t.Fatal("raw error body leaked")
			}
			if _, err := transport.request(context.Background(), "POST", "/completion", make([]byte, MaxRequestBytes+1)); err == nil {
				t.Fatal("large request sent")
			}
		})
	}
}

func TestUDSTransportCancellation(t *testing.T) {
	_, transport := localTestServer(t, func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() })
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := transport.request(ctx, "GET", "/health", nil); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("lost cancellation: %v", err)
	}
}

func TestUDSTransportClassifiesRequestByteOverflowBeforeSending(t *testing.T) {
	var requests atomic.Int32
	_, transport := localTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		_, _ = io.Copy(io.Discard, r.Body)
		_, _ = w.Write([]byte(`{}`))
	})
	for _, path := range []string{"/apply-template", "/tokenize", "/completion"} {
		if _, err := transport.request(context.Background(), "POST", path, make([]byte, MaxRequestBytes+1)); !errors.Is(err, errRequestTooLarge) {
			t.Fatalf("path=%s err=%v", path, err)
		}
	}
	if requests.Load() != 0 {
		t.Fatal("oversized request reached the server")
	}
	if _, err := transport.request(context.Background(), "POST", "/apply-template", make([]byte, MaxRequestBytes)); err != nil || requests.Load() != 1 {
		t.Fatalf("exact limit rejected: requests=%d err=%v", requests.Load(), err)
	}
}
