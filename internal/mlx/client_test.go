package mlx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/natsuki0413/commiter-cli/internal/llm"
)

func TestGenerateKeepsStdoutProtocolSeparateFromStderr(t *testing.T) {
	helper := shellHelper(t, `IFS= read -r _
printf 'model loading is diagnostic\n' >&2
printf '%s\n' '{"ok":true,"stop_reason":"completed","generated_json":"{}","model":"qwen3.5:4b","runtime":"mlx"}'
`)
	response, err := NewClient(helper).Generate(context.Background(), validRequest())
	if err != nil {
		t.Fatal(err)
	}
	if response.GeneratedJSON != "{}" || response.Model != "qwen3.5:4b" || response.Runtime != "mlx" {
		t.Fatalf("response = %+v", response)
	}
}

func TestGenerateRejectsNonCompletedStopStateAndPartialJSON(t *testing.T) {
	for _, reason := range []StopReason{StopReasonMaxTokens, StopReasonGrammar, StopReasonInternal} {
		t.Run(string(reason), func(t *testing.T) {
			helper := shellHelper(t, "IFS= read -r _\nprintf '%s\\n' "+shellQuote(fmt.Sprintf(
				`{"ok":false,"stop_reason":%q,"generated_json":"{\"partial\":true}"}`, reason))+"\n")
			response, err := NewClient(helper).Generate(context.Background(), validRequest())
			if response.StopReason != reason {
				t.Fatalf("response = %+v", response)
			}
			var failure *Error
			if !errors.As(err, &failure) || failure.Kind != FailureStopState || failure.StopReason != reason {
				t.Fatalf("error = %#v", err)
			}
		})
	}
}

func TestGenerateRejectsMalformedOrTrailingResponse(t *testing.T) {
	tests := []struct {
		name   string
		output string
	}{
		{name: "malformed", output: "not-json\n"},
		{name: "missing newline", output: "{\"ok\":true,\"stop_reason\":\"completed\",\"generated_json\":\"{}\"}"},
		{name: "trailing line", output: "{\"ok\":true,\"stop_reason\":\"completed\",\"generated_json\":\"{}\"}\n{}\n"},
		{name: "unknown field", output: "{\"ok\":true,\"stop_reason\":\"completed\",\"generated_json\":\"{}\",\"prompt\":\"secret\"}\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			helper := shellHelper(t, "IFS= read -r _\nprintf '%s' "+shellQuote(test.output)+"\n")
			_, err := NewClient(helper).Generate(context.Background(), validRequest())
			var failure *Error
			if !errors.As(err, &failure) || failure.Kind != FailureProtocol {
				t.Fatalf("error = %#v", err)
			}
		})
	}
}

func TestGenerateRejectsInvalidStopReasonAsProtocol(t *testing.T) {
	helper := shellHelper(t, `IFS= read -r _
printf '%s\n' '{"ok":true,"stop_reason":"unknown","generated_json":"{}"}'
`)
	_, err := NewClient(helper).Generate(context.Background(), validRequest())
	var failure *Error
	if !errors.As(err, &failure) || failure.Kind != FailureProtocol {
		t.Fatalf("error = %#v", err)
	}
}

func TestGenerateRejectsOversizedRequestBeforeStartingHelper(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "started")
	helper := shellHelper(t, fmt.Sprintf("printf started > %s\nprintf '%s\\n' '{\"ok\":true,\"stop_reason\":\"completed\",\"generated_json\":\"{}\"}'\n", shellQuote(marker), shellQuote("{\"ok\":true,\"stop_reason\":\"completed\",\"generated_json\":\"{}\"}")))
	client := NewClient(helper)
	client.MaxRequestBytes = 128
	request := validRequest()
	request.Messages = []llm.Message{{Role: "user", Content: strings.Repeat("x", 512)}}
	_, err := client.Generate(context.Background(), request)
	var failure *Error
	if !errors.As(err, &failure) || failure.Kind != FailureOversized {
		t.Fatalf("error = %#v", err)
	}
	if _, statErr := os.Stat(marker); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("helper started unexpectedly: %v", statErr)
	}
}

func TestGenerateRejectsOversizedResponse(t *testing.T) {
	helper := shellHelper(t, `
while :; do printf '%*s' 2048 ''; done
`)
	client := NewClient(helper)
	client.MaxResponseBytes = 128
	_, err := client.Generate(context.Background(), validRequest())
	var failure *Error
	if !errors.As(err, &failure) || failure.Kind != FailureOversized {
		t.Fatalf("error = %#v", err)
	}
}

func TestGenerateClassifiesHelperCrash(t *testing.T) {
	helper := shellHelper(t, "exit 17\n")
	_, err := NewClient(helper).Generate(context.Background(), validRequest())
	var failure *Error
	if !errors.As(err, &failure) || failure.Kind != FailureCrash || failure.ExitCode != 17 {
		t.Fatalf("error = %#v", err)
	}
}

func TestGenerateTimeoutKillsProcessGroup(t *testing.T) {
	dir := t.TempDir()
	pidFile := filepath.Join(dir, "child.pid")
	survivedFile := filepath.Join(dir, "survived")
	helper := shellHelper(t, fmt.Sprintf(`
trap 'exit 0' TERM
sh -c 'trap "" TERM; printf "%%s" "$$" > %s; sleep 30; printf survived > %s' &
printf started > %s
wait
`, pidFile, survivedFile, filepath.Join(dir, "started")))
	client := NewClient(helper)
	client.TerminateGrace = 50 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	response, err := client.Generate(ctx, validRequest())
	var failure *Error
	if !errors.As(err, &failure) || failure.Kind != FailureTimeout || failure.StopReason != StopReasonTimeout {
		t.Fatalf("error = %#v", err)
	}
	if response.OK || response.StopReason != StopReasonTimeout || response.GeneratedJSON != "" {
		t.Fatalf("response = %+v", response)
	}
	if !waitForTestPath(pidFile, time.Second) {
		t.Fatal("child did not start")
	}
	time.Sleep(150 * time.Millisecond)
	if _, statErr := os.Stat(survivedFile); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("descendant survived cancellation: %v", statErr)
	}
	data, readErr := os.ReadFile(pidFile)
	if readErr != nil {
		t.Fatal(readErr)
	}
	var pid int
	if _, scanErr := fmt.Sscanf(string(data), "%d", &pid); scanErr != nil {
		t.Fatal(scanErr)
	}
	if !waitForTestProcessExit(pid, time.Second) {
		t.Fatalf("descendant process %d survived cancellation", pid)
	}
}

func TestGenerateCancellationIsClassified(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "started")
	helper := shellHelper(t, fmt.Sprintf("printf started > %s\nsleep 30\n", shellQuote(marker)))
	client := NewClient(helper)
	ctx, cancel := context.WithCancel(context.Background())
	type resultValue struct {
		response Response
		err      error
	}
	result := make(chan resultValue, 1)
	go func() {
		response, err := client.Generate(ctx, validRequest())
		result <- resultValue{response: response, err: err}
	}()
	if !waitForTestPath(marker, time.Second) {
		t.Fatal("helper did not start")
	}
	cancel()
	got := <-result
	var failure *Error
	if !errors.As(got.err, &failure) || failure.Kind != FailureCancelled || failure.StopReason != StopReasonCancelled {
		t.Fatalf("error = %#v", got.err)
	}
	if got.response.OK || got.response.StopReason != StopReasonCancelled || got.response.GeneratedJSON != "" {
		t.Fatalf("response = %+v", got.response)
	}
}

func TestGenerateAlreadyEndedContextHasStopReason(t *testing.T) {
	for _, test := range []struct {
		name        string
		makeContext func() (context.Context, context.CancelFunc)
		kind        FailureKind
		reason      StopReason
	}{
		{name: "cancelled", makeContext: func() (context.Context, context.CancelFunc) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			return ctx, cancel
		}, kind: FailureCancelled, reason: StopReasonCancelled},
		{name: "timeout", makeContext: func() (context.Context, context.CancelFunc) {
			return context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
		}, kind: FailureTimeout, reason: StopReasonTimeout},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := test.makeContext()
			defer cancel()
			marker := filepath.Join(t.TempDir(), "started")
			helper := shellHelper(t, "printf started > "+shellQuote(marker)+"\n")
			response, err := NewClient(helper).Generate(ctx, validRequest())
			var failure *Error
			if !errors.As(err, &failure) || failure.Kind != test.kind || failure.StopReason != test.reason {
				t.Fatalf("error = %#v", err)
			}
			if response.OK || response.StopReason != test.reason || response.GeneratedJSON != "" {
				t.Fatalf("response = %+v", response)
			}
			if _, statErr := os.Stat(marker); !errors.Is(statErr, os.ErrNotExist) {
				t.Fatalf("helper started unexpectedly: %v", statErr)
			}
		})
	}
}

func TestMarshalRequestRequiresValidSchema(t *testing.T) {
	request := validRequest()
	request.Schema = json.RawMessage("not-json")
	if _, err := marshalRequest(request, DefaultMaxRequestBytes); err == nil {
		t.Fatal("invalid schema was accepted")
	}
}

func TestMarshalRequestPreservesLocalModelPath(t *testing.T) {
	request := validRequest()
	request.Model = "local-qwen"
	request.ModelPath = "/models/local-qwen"

	data, err := marshalRequest(request, DefaultMaxRequestBytes)
	if err != nil {
		t.Fatal(err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(data), &decoded); err != nil {
		t.Fatal(err)
	}
	if got, ok := decoded["model_path"].(string); !ok || got != request.ModelPath {
		t.Fatalf("model_path = %#v", decoded["model_path"])
	}
}

func validRequest() Request {
	return Request{
		Schema:       json.RawMessage(`{"type":"object"}`),
		Messages:     []llm.Message{{Role: "user", Content: "produce a plan"}},
		OutputTokens: 128,
	}
}

func shellHelper(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "helper.sh")
	content := "#!/bin/sh\nset -eu\n" + body
	if err := os.WriteFile(path, []byte(content), 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func waitForTestPath(path string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

func waitForTestProcessExit(pid int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if err := syscall.Kill(pid, 0); errors.Is(err, syscall.ESRCH) {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}
