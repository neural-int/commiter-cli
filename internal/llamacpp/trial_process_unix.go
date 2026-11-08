//go:build darwin || linux

package llamacpp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

type trialLog struct {
	mu                     sync.Mutex
	failed                 bool
	constraintFailure      string
	pending                []byte
	modelLoads             int
	metal                  bool
	offloaded, totalLayers int
}

func (l *trialLog) Write(data []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	n := len(data)
	l.pending = append(l.pending, data...)
	// Classify before consuming complete lines or rotating an oversized partial
	// line. Keep the failure sticky without retaining raw process history.
	if strings.Contains(string(l.pending), "llg error:") || strings.Contains(string(l.pending), "llg tokenizer error:") || strings.Contains(string(l.pending), "commiter: LLGuidance ") {
		l.failed = true
	}
	for {
		newline := strings.IndexByte(string(l.pending), '\n')
		if newline < 0 {
			break
		}
		line := string(l.pending[:newline])
		l.pending = l.pending[newline+1:]
		for _, operation := range []string{"compile", "mask", "consume", "reset", "clone", "state", "tokenizer", "tokenizer clone"} {
			if strings.Contains(line, "commiter: LLGuidance "+operation+" failed; stopping server") {
				l.constraintFailure = operation
			}
		}
		if strings.Contains(line, "loaded meta data with") {
			l.modelLoads++
		}
		// The fixed Metal backend names its device MTL0 (not "Metal").
		if strings.Contains(line, "using device MTL") {
			l.metal = true
		}
		if _, rest, ok := strings.Cut(line, "offloaded "); ok {
			_, _ = fmt.Sscanf(rest, "%d/%d layers to GPU", &l.offloaded, &l.totalLayers)
		}
	}
	if len(l.pending) > 64<<10 {
		l.pending = append([]byte(nil), l.pending[len(l.pending)-(64<<10):]...)
	}
	return n, nil
}
func (l *trialLog) grammarFailed() bool { l.mu.Lock(); defer l.mu.Unlock(); return l.failed }

func (l *trialLog) observation() TrialObservation {
	l.mu.Lock()
	defer l.mu.Unlock()
	return TrialObservation{ModelLoads: l.modelLoads, MetalDevice: l.metal, OffloadedLayers: l.offloaded, TotalLayers: l.totalLayers, ConstraintFailure: l.constraintFailure}
}

type trialProcess struct {
	command   *exec.Cmd
	directory string
	identity  os.FileInfo
	transport *localTransport
	logs      trialLog
	done      chan struct{}
	waitErr   error
	once      sync.Once
	closeErr  error
	cancel    context.CancelFunc
	ctx       context.Context
}

func startTrialProcess(ctx context.Context, executable, model string) (result *trialProcess, err error) {
	if !filepath.IsAbs(executable) || !filepath.IsAbs(model) {
		return nil, errTrial
	}
	info, e := os.Lstat(executable)
	if e != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0111 == 0 {
		return nil, errTrial
	}
	dir, e := os.MkdirTemp("/tmp", "c190-")
	if e != nil {
		return nil, errTrial
	}
	identity, e := os.Lstat(dir)
	if e != nil {
		_ = os.RemoveAll(dir)
		return nil, errTrial
	}
	child, cancel := context.WithCancel(ctx)
	p := &trialProcess{directory: dir, identity: identity, done: make(chan struct{}), ctx: child, cancel: cancel}
	defer func() {
		if err != nil {
			err = errors.Join(err, p.close())
		}
	}()
	socket := filepath.Join(dir, "server.sock")
	// This revision maps model/Metal GGML info to trace level 4. Its raw logs
	// stay in the bounded private buffer; only classified observations leave it.
	args := []string{executable, "--model", model, "--host", socket, "--ctx-size", "65536", "--parallel", "1", "--n-gpu-layers", "all", "--no-context-shift", "--cache-ram", "0", "--no-cache-prompt", "--no-slots", "--no-webui", "--offline", "--jinja", "--reasoning", "off", "--threads", "4", "--threads-batch", "4", "--no-warmup", "--fit", "off", "--log-verbosity", "4"}
	// A constant shell prelude applies the child's umask without mutating the
	// multi-threaded Go parent's global umask. exec preserves the observed PID.
	p.command = exec.Command("/bin/sh", append([]string{"-c", `umask 077; exec "$@"`, "commiter-llamacpp-trial"}, args...)...)
	p.command.Env = []string{"PATH=/usr/bin:/bin", "LC_ALL=C"}
	p.command.Dir = dir
	p.command.Stdout = &p.logs
	p.command.Stderr = &p.logs
	p.command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if e := p.command.Start(); e != nil {
		close(p.done)
		return nil, errTrial
	}
	go func() { p.waitErr = p.command.Wait(); close(p.done) }()
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-p.done:
			return nil, errors.New("llamacpp: trial server exited before readiness")
		case <-ticker.C:
			if p.transport == nil {
				// bind(2) creates a Unix socket with 0777 masked by the child's
				// umask (0700 here). Remove its execute bit before any connection.
				if info, e := os.Lstat(socket); e == nil {
					if info.Mode()&os.ModeSocket == 0 || !localOwner(info) || (info.Mode().Perm() != 0700 && info.Mode().Perm() != 0600) {
						return nil, errTransport
					}
					if e := os.Chmod(socket, 0600); e != nil {
						return nil, errTransport
					}
					after, e := os.Lstat(socket)
					if e != nil || !os.SameFile(info, after) {
						return nil, errTransport
					}
				}
				transport, e := newLocalTransport(socket, p.command.Process.Pid)
				if e != nil {
					continue
				}
				p.transport = transport
			}
			probe, stop := context.WithTimeout(child, 200*time.Millisecond)
			data, e := p.transport.request(probe, http.MethodGet, "/health", nil)
			stop()
			var health struct {
				Status string `json:"status"`
			}
			if e == nil && json.Unmarshal(data, &health) == nil && health.Status == "ok" {
				go func() {
					select {
					case <-child.Done():
						_ = p.close()
					case <-p.done:
					}
				}()
				return p, nil
			}
		}
	}
}

func (p *trialProcess) call(ctx context.Context, path string, request any, response any) error {
	if err := p.ctx.Err(); err != nil {
		return err
	}
	select {
	case <-p.done:
		return errors.New("llamacpp: trial process exited")
	default:
	}
	data, err := json.Marshal(request)
	if err != nil {
		return errTrial
	}
	data, err = p.transport.request(ctx, http.MethodPost, path, data)
	if err != nil {
		return err
	}
	if _, err := strictJSON(data); err != nil {
		return errTrial
	}
	if err := json.Unmarshal(data, response); err != nil {
		return errTrial
	}
	return nil
}

func (p *trialProcess) close() error {
	p.once.Do(func() {
		p.cancel()
		if p.transport != nil {
			p.transport.close()
		}
		if p.command != nil && p.command.Process != nil {
			select {
			case <-p.done:
				if p.waitErr != nil {
					p.closeErr = errors.New("llamacpp: trial server exited abnormally")
				}
			default:
				_ = syscall.Kill(-p.command.Process.Pid, syscall.SIGTERM)
				select {
				case <-p.done:
				case <-time.After(time.Second):
					_ = syscall.Kill(-p.command.Process.Pid, syscall.SIGKILL)
					select {
					case <-p.done:
					case <-time.After(time.Second):
						p.closeErr = errors.New("llamacpp: trial process cleanup timed out")
						return
					}
				}
			}
		}
		info, err := os.Lstat(p.directory)
		if err != nil || !os.SameFile(info, p.identity) || !localOwner(info) {
			p.closeErr = errors.Join(p.closeErr, errTrial)
			return
		}
		if err := os.RemoveAll(p.directory); err != nil {
			p.closeErr = errors.Join(p.closeErr, errTrial)
		}
	})
	return p.closeErr
}
