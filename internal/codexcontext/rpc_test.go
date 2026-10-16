package codexcontext

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// A real subprocess exercises pipe cancellation and byte caps without Codex,
// authentication, network access or private user history.
func TestRPCSubprocessHelper(t *testing.T) {
	mode := os.Getenv("COMMITER_RPC_TEST_HELPER")
	if mode == "" {
		return
	}
	s := bufio.NewScanner(os.Stdin)
	for s.Scan() {
		var request struct {
			ID     int    `json:"id"`
			Method string `json:"method"`
		}
		_ = json.Unmarshal(s.Bytes(), &request)
		if request.ID == 0 {
			continue
		}
		switch mode {
		case "hang":
			time.Sleep(time.Minute)
		case "oversize":
			fmt.Println(strings.Repeat("x", MaxReceiveBytes+100))
		case "error":
			fmt.Printf("{\"id\":%d,\"error\":{\"message\":\"PRIVATE\"}}\n", request.ID)
		case "approval":
			fmt.Println(`{"id":90,"method":"item/commandExecution/requestApproval","params":{}}`)
		case "malformed":
			fmt.Println(`{"id":`)
		default:
			fmt.Println(`{"method":"notification","params":{}}`)
			fmt.Printf("{\"id\":%d,\"result\":{\"data\":[]}}\n", request.ID)
		}
	}
	os.Exit(0)
}

func helperProcess(t *testing.T, mode string, ctx context.Context) *process {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(ctx, exe, "-test.run=^TestRPCSubprocessHelper$")
	cmd.Env = append(os.Environ(), "COMMITER_RPC_TEST_HELPER="+mode)
	p, err := startProcess(ctx, cmd)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(p.close)
	return p
}

func TestRPCTransportFailsClosedAndDoesNotLeakErrors(t *testing.T) {
	for _, mode := range []string{"ok", "oversize", "error", "approval", "malformed"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			p := helperProcess(t, mode, ctx)
			var result map[string]any
			err := p.call("thread/list", map[string]any{}, &result)
			if (err == nil) != (mode == "ok") {
				t.Fatalf("mode=%s err=%v", mode, err)
			}
			if err != nil && strings.Contains(err.Error(), "PRIVATE") {
				t.Fatal("server error leaked")
			}
		})
	}
}

func TestRPCCancellationUnblocksHangingChild(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	p := helperProcess(t, "hang", ctx)
	started := time.Now()
	var result any
	if p.call("thread/list", nil, &result) == nil {
		t.Fatal("hanging server succeeded")
	}
	if time.Since(started) > time.Second {
		t.Fatal("deadline did not unblock transport")
	}
}

func TestMissingCodexFallsBack(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	result := Collect(context.Background(), Request{Root: t.TempDir(), Start: time.Unix(1, 0), End: time.Now(), Targets: []Target{{Path: "a.go"}}})
	if result.Status != "unavailable" || len(result.Records) != 0 {
		t.Fatalf("result=%+v", result)
	}
}
