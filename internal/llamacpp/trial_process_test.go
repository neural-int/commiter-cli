//go:build darwin || linux

package llamacpp

import (
	"context"
	"errors"
	"fmt"
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
	for _, marker := range []string{"llg error:", "commiter: LLGuidance mask failed; stopping server"} {
		t.Run(marker, func(t *testing.T) {
			log := &trialLog{}
			_, _ = log.Write([]byte(marker[:5]))
			_, _ = log.Write([]byte(marker[5:] + "\n"))
			_, _ = log.Write([]byte(strings.Repeat("x", 128<<10)))
			if !log.grammarFailed() {
				t.Fatal("grammar failure was lost after log rotation")
			}
			if strings.HasPrefix(marker, "commiter:") && log.observation().ConstraintFailure != "mask" {
				t.Fatal("classified failure operation was lost after log rotation")
			}
			if len(log.pending) > 64<<10 {
				t.Fatal("unbounded diagnostic line")
			}
			_, _ = log.Write([]byte("\n"))
			if len(log.pending) != 0 {
				t.Fatal("raw completed log lines retained")
			}
		})
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
		if r.URL.Path == "/props" {
			_, _ = fmt.Fprintf(w, `{"default_generation_settings":{"n_ctx":65536},"total_slots":1,"model_path":%q,"build_info":%q}`, model, TrialLlamaBuild)
			return
		}
		http.Error(w, "not found", 404)
	}), ReadHeaderTimeout: time.Second}
	if strings.HasSuffix(model, "delayed.gguf") {
		go func() {
			for {
				if _, err := os.Stat(filepath.Join(filepath.Dir(socket), "release-startup-log")); err == nil {
					_, _ = fmt.Fprintln(os.Stderr, "loaded meta data with synthetic fields\nusing device MTL0\noffloaded 33/33 layers to GPU")
					return
				}
				time.Sleep(5 * time.Millisecond)
			}
		}()
	}
	_ = server.Serve(listener)
}

func TestTrialStartupObservationWaitsForDelayedLogs(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	p, err := startTrialProcess(ctx, trialTestExecutable(t), "/synthetic/delayed.gguf")
	if err != nil {
		t.Fatal(err)
	}
	defer p.close()
	if _, err := p.transport.request(ctx, http.MethodGet, "/props", nil); err != nil {
		t.Fatal(err)
	}
	if p.logs.observation().ModelLoads != 0 {
		t.Fatal("startup logs appeared before the explicit release")
	}
	result := make(chan error, 1)
	go func() { result <- p.waitForStartupObservation(ctx) }()
	if err := os.WriteFile(filepath.Join(p.directory, "release-startup-log"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-result:
		if err != nil {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatal("delayed startup logs were not observed")
	}
	observation := p.logs.observation()
	if observation.ModelLoads != 1 || !observation.MetalDevice || observation.OffloadedLayers != 33 || observation.TotalLayers != 33 {
		t.Fatalf("incomplete startup accepted: %+v", observation)
	}
}

func TestTrialStartupObservationMissingLogsHonorsDeadlineAndReapsChild(t *testing.T) {
	parent, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	p, err := startTrialProcess(parent, trialTestExecutable(t), "/synthetic/model.gguf")
	if err != nil {
		t.Fatal(err)
	}
	defer p.close()
	ctx, stop := context.WithTimeout(parent, 50*time.Millisecond)
	defer stop()
	if err := p.waitForStartupObservation(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("missing logs did not preserve deadline: %v", err)
	}
	if err := p.close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-p.done:
	default:
		t.Fatal("child was not reaped after observation timeout")
	}
	if _, err := os.Lstat(p.directory); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("startup observation timeout left resources")
	}
}

func TestTrialStartupObservationRejectsInvalidOrInterruptedStartup(t *testing.T) {
	for _, mode := range []string{"cancel", "process cancel", "exit", "duplicate load", "partial offload", "constraint failure", "patched constraint failure"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			child, stop := context.WithCancel(ctx)
			defer stop()
			p := &trialProcess{ctx: child, done: make(chan struct{})}
			_, _ = p.logs.Write([]byte("loaded meta data with synthetic fields\nusing device MTL0\noffloaded 33/33 layers to GPU\n"))
			switch mode {
			case "cancel":
				cancel()
			case "process cancel":
				stop()
			case "exit":
				close(p.done)
			case "duplicate load":
				_, _ = p.logs.Write([]byte("loaded meta data with another model\n"))
			case "partial offload":
				_, _ = p.logs.Write([]byte("offloaded 32/33 layers to GPU\n"))
			case "constraint failure":
				_, _ = p.logs.Write([]byte("llg error: synthetic\n"))
			case "patched constraint failure":
				_, _ = p.logs.Write([]byte("commiter: LLGuidance mask failed; stopping server\n"))
			}
			err := p.waitForStartupObservation(ctx)
			if err == nil || errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("invalid startup was not rejected immediately: %v", err)
			}
			if (mode == "cancel" || mode == "process cancel") && !errors.Is(err, context.Canceled) {
				t.Fatalf("cancellation cause lost: %v", err)
			}
		})
	}
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
