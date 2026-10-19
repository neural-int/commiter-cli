package codexcontext

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os/exec"
	"strconv"
	"time"
)

var errUnavailable = errors.New("Codex history unavailable")

type caller interface {
	call(string, any, any) error
}

type process struct {
	cmd     *exec.Cmd
	in      io.WriteCloser
	out     io.ReadCloser
	scanner *bufio.Scanner
	reader  *io.LimitedReader
	stop    func() bool
	seq     int
}

func start(ctx context.Context, root string) (*process, error) {
	cmd := exec.CommandContext(ctx, "codex", "app-server", "--stdio", "-c", "analytics.enabled=false")
	cmd.Dir = root
	return startProcess(ctx, cmd)
}

func startProcess(ctx context.Context, cmd *exec.Cmd) (*process, error) {
	cmd.WaitDelay = 100 * time.Millisecond
	in, err := cmd.StdinPipe()
	if err != nil {
		return nil, errUnavailable
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		in.Close()
		return nil, errUnavailable
	}
	// stderr is deliberately discarded: diagnostics may contain private history.
	if err := cmd.Start(); err != nil {
		in.Close()
		out.Close()
		return nil, errUnavailable
	}
	p := &process{cmd: cmd, in: in, out: out, reader: &io.LimitedReader{R: out, N: MaxReceiveBytes + 1}}
	p.scanner = bufio.NewScanner(p.reader)
	p.scanner.Buffer(make([]byte, 4096), MaxReceiveBytes+1)
	p.stop = context.AfterFunc(ctx, func() { in.Close(); out.Close() })
	return p, nil
}

func (p *process) close() {
	p.stop()
	p.in.Close()
	p.out.Close()
	// Kill only our transport process, never a shared Codex daemon/process group.
	_ = p.cmd.Process.Kill()
	_ = p.cmd.Wait()
}

func (p *process) call(method string, params any, result any) error {
	p.seq++
	if err := json.NewEncoder(p.in).Encode(map[string]any{"id": p.seq, "method": method, "params": params}); err != nil {
		return errUnavailable
	}
	for p.scanner.Scan() {
		if p.reader.N <= 1 {
			return errUnavailable
		}
		var reply struct {
			ID     json.RawMessage `json:"id"`
			Result json.RawMessage `json:"result"`
			Error  json.RawMessage `json:"error"`
			Method string          `json:"method"`
		}
		if json.Unmarshal(p.scanner.Bytes(), &reply) != nil {
			return errUnavailable
		}
		if reply.Method != "" && len(reply.ID) > 0 {
			return errUnavailable
		} // no approvals or server actions
		if string(reply.ID) != strconv.Itoa(p.seq) {
			continue
		}
		if len(reply.Error) > 0 && string(reply.Error) != "null" {
			return errUnavailable
		}
		if len(reply.Result) == 0 || json.Unmarshal(reply.Result, result) != nil {
			return errUnavailable
		}
		return nil
	}
	return errUnavailable
}

func (p *process) initialize() error {
	var result map[string]json.RawMessage
	if err := p.call("initialize", map[string]any{"clientInfo": map[string]string{"name": "commiter", "version": "1"}, "capabilities": map[string]bool{"experimentalApi": true}}, &result); err != nil {
		return err
	}
	return json.NewEncoder(p.in).Encode(map[string]string{"method": "initialized"})
}
