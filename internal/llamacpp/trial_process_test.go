//go:build darwin || linux

package llamacpp

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func trialTestExecutable(t *testing.T) string {
	t.Helper()
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "fake-server")
	quoted := "'" + strings.ReplaceAll(binary, "'", "'\"'\"'") + "'"
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexec "+quoted+" -test.run=^TestTrialServerHelper$ -- \"$@\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestTrialLogKeepsFailureAfterBoundedBufferRotation(t *testing.T) {
	log := &trialLog{}
	_, _ = log.Write([]byte("llg err"))
	_, _ = log.Write([]byte("or: synthetic\n"))
	_, _ = log.Write([]byte(strings.Repeat("x", 128<<10)))
	if !log.grammarFailed() {
		t.Fatal("grammar failure was lost after log rotation")
	}
	if len(log.pending) > 64<<10 {
		t.Fatal("unbounded diagnostic line")
	}
	_, _ = log.Write([]byte("\n"))
	if len(log.pending) != 0 {
		t.Fatal("raw completed log lines retained")
	}
}

func TestTrialServerHelper(t *testing.T) {
	model, socket := "", ""
	for i := 0; i+1 < len(os.Args); i++ {
		if os.Args[i] == "--host" {
			socket = os.Args[i+1]
		}
		if os.Args[i] == "--model" {
			model = os.Args[i+1]
		}
	}
	if socket == "" {
		return
	}
	if os.Getenv("HF_TOKEN") != "" || os.Getenv("LLAMA_ARG_HOST") != "" || os.Getenv("HTTP_PROXY") != "" {
		os.Exit(9)
	}
	if strings.HasSuffix(model, "hang.gguf") {
		for {
			time.Sleep(time.Second)
		}
	}
	listener, err := net.Listen("unix", socket)
	if err != nil {
		os.Exit(8)
	}
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			_, _ = w.Write([]byte(`{"status":"ok"}`))
			return
		}
		http.Error(w, "not found", 404)
	}), ReadHeaderTimeout: time.Second}
	_ = server.Serve(listener)
}

func TestTrialProcessReadyAndIdempotentCleanup(t *testing.T) {
	t.Setenv("HF_TOKEN", "SYNTHETIC_TEST_SENTINEL")
	t.Setenv("LLAMA_ARG_HOST", "SYNTHETIC_TEST_SENTINEL")
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:1")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	p, err := startTrialProcess(ctx, trialTestExecutable(t), "/synthetic/model.gguf")
	if err != nil {
		t.Fatal(err)
	}
	directory := p.directory
	if err := p.close(); err != nil {
		t.Fatal(err)
	}
	if err := p.close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(directory); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("trial resources remain after close")
	}
}

func TestTrialProcessCancellationReapsChild(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	p, err := startTrialProcess(ctx, trialTestExecutable(t), "/synthetic/model.gguf")
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	cancel()
	if err := p.close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-p.done:
	case <-time.After(3 * time.Second):
		t.Fatal("child survived cancellation")
	}
	if _, err := os.Lstat(p.directory); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("cancelled cycle left resources")
	}
}

func TestTrialProcessReadinessTimeoutAndStartFailure(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if _, err := startTrialProcess(ctx, trialTestExecutable(t), "/synthetic/hang.gguf"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("readiness timeout lost: %v", err)
	}
	executable := filepath.Join(t.TempDir(), "invalid-server")
	if err := os.WriteFile(executable, []byte("#!/missing/interpreter\n"), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := startTrialProcess(context.Background(), executable, "/synthetic/model.gguf"); err == nil {
		t.Fatal("failed process start accepted")
	}
}
